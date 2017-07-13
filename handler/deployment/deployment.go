// Package deployment provides functionality for planning and running deployment jobs.
package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/ONSdigital/dp-ci/awdry/engine"
	"github.com/ONSdigital/go-ns/log"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/slimsag/untargz"
)

const (
	// DefaultAllocationTimeout is the default time to wait for an allocation to complete.
	DefaultAllocationTimeout = time.Second * 300
	// DefaultEvaluationTimeout is the default time to wait for an evaluation to complete.
	DefaultEvaluationTimeout = time.Second * 120
)

const (
	allocURL = "%s/v1/evaluation/%s/allocations"
	evalURL  = "%s/v1/evaluation/%s"
	planURL  = "%s/v1/job/%s/plan"
	runURL   = "%s/v1/jobs"
)

const (
	statusComplete = "complete"
	statusPending  = "pending"
	statusRunning  = "running"
)

type payload struct {
	Job *api.Job
}

// Deployment represents a deployment.
type Deployment struct {
	client   *s3.S3
	root     string
	endpoint string
	timeout  *TimeoutConfig
}

// New returns a new deployment.
func New(c *Config) (*Deployment, error) {
	a, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}
	if c.Timeout != nil && c.Timeout.Allocation < 1 {
		c.Timeout.Allocation = DefaultAllocationTimeout
	}
	if c.Timeout != nil && c.Timeout.Evaluation < 1 {
		c.Timeout.Evaluation = DefaultEvaluationTimeout
	}
	if c.Timeout == nil {
		c.Timeout = &TimeoutConfig{DefaultAllocationTimeout, DefaultEvaluationTimeout}
	}

	return &Deployment{
		client:   s3.New(a, aws.Regions[c.Region], http.DefaultClient),
		root:     c.DeploymentRoot,
		endpoint: c.NomadEndpoint,
		timeout:  c.Timeout,
	}, nil
}

// Handler handles deployment messages that are delegated by the engine.
func (d *Deployment) Handler(ctx context.Context, msg *engine.Message) error {
	b, err := d.client.Bucket(msg.Bucket).Get(msg.Artifact)
	if err != nil {
		return err
	}
	if err := untargz.Extract(bytes.NewReader(b), fmt.Sprintf("%s/%s", d.root, msg.Service), nil); err != nil {
		return err
	}
	if err := d.plan(msg); err != nil {
		return err
	}
	if err := d.run(ctx, msg); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) plan(msg *engine.Message) error {
	log.Trace("planning job", log.Data{"msg": msg, "service": msg.Service})

	var res api.JobPlanResponse
	if err := d.post(fmt.Sprintf(planURL, d.endpoint, msg.Service), msg, &res); err != nil {
		return err
	}
	if len(res.Warnings) == 0 && res.FailedTGAllocs == nil {
		return nil
	}
	if len(res.Warnings) > 0 {
		return &PlanError{service: msg.Service, warnings: res.Warnings}
	}
	j, err := json.Marshal(res.FailedTGAllocs)
	if err != nil {
		return err
	}
	return &PlanError{errors: string(j), service: msg.Service}
}

func (d *Deployment) run(ctx context.Context, msg *engine.Message) error {
	log.Trace("running job", log.Data{"msg": msg, "service": msg.Service})

	var res api.JobRegisterResponse
	if err := d.post(fmt.Sprintf(runURL, d.endpoint), msg, &res); err != nil {
		return err
	}
	if err := d.monitor(ctx, res.EvalID); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) monitor(ctx context.Context, id string) error {
	if err := d.evaluation(ctx, id); err != nil {
		return err
	}
	if err := d.allocations(ctx, id); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) allocations(ctx context.Context, id string) error {
	ticker := time.Tick(time.Second * 1)
	timeout := time.After(d.timeout.Allocation)

	for {
		select {
		case <-ctx.Done():
			log.Info("bailing on deployment allocations", log.Data{"evaluation": id})
			return &AllocationAbortedError{evaluationID: id}
		case <-timeout:
			return &TimeoutError{action: "allocation"}
		case <-ticker:
			var allocations []api.Allocation
			if err := d.get(fmt.Sprintf(allocURL, d.endpoint, id), &allocations); err != nil {
				return err
			}
			pending, running := sumAllocations(&allocations)
			if pending > 0 {
				log.Trace("allocations still pending", log.Data{"evaluation": id, "pending": pending, "total": len(allocations)})
				continue
			}
			if running == len(allocations) {
				log.Trace("all allocations running", log.Data{"evaluation": id, "running": running, "total": len(allocations)})
				return nil
			}
			return &AllocationError{pending, running, len(allocations)}
		default:
		}
	}
}

func sumAllocations(allocations *[]api.Allocation) (pending, running int) {
	for _, allocation := range *allocations {
		switch allocation.ClientStatus {
		case statusRunning:
			running++
		case statusPending:
			pending++
		}
	}
	return
}

func (d *Deployment) evaluation(ctx context.Context, id string) error {
	ticker := time.Tick(time.Second * 1)
	timeout := time.After(d.timeout.Evaluation)

	for {
		select {
		case <-ctx.Done():
			log.Info("bailing on deployment evaluation", log.Data{"evaluation": id})
			return &EvaluationAbortedError{id: id}
		case <-timeout:
			return &TimeoutError{action: "evaluation"}
		case <-ticker:
			var evaluation api.Evaluation
			if err := d.get(fmt.Sprintf(evalURL, d.endpoint, id), &evaluation); err != nil {
				return err
			}
			if evaluation.Status == statusPending {
				log.Trace("waiting for evaluation to be scheduled", log.Data{"id": evaluation.ID})
				continue
			}
			if evaluation.Status != statusComplete {
				return &EvaluationError{id: evaluation.ID}
			}
			log.Trace("evaluation complete", log.Data{"id": evaluation.ID})
			if len(evaluation.NextEval) == 0 {
				return nil
			}
			log.Info("waiting for next evaluation", log.Data{"id": evaluation.ID, "next evaluation": evaluation.NextEval})
			return d.monitor(ctx, evaluation.NextEval)
		default:
		}
	}
}

func (d *Deployment) get(url string, v interface{}) error {
	r, err := http.DefaultClient.Get(url)
	if err != nil {
		return err
	}
	return unmarshalAPIResponse(r, &v)
}

func (d *Deployment) post(url string, msg *engine.Message, v interface{}) error {
	j, err := jsonFrom(fmt.Sprintf("%s/%s/%s.nomad", d.root, msg.Service, msg.Service))
	if err != nil {
		return err
	}
	r, err := http.DefaultClient.Post(url, "application/json", bytes.NewReader(j))
	if err != nil {
		return err
	}
	return unmarshalAPIResponse(r, v)
}

func unmarshalAPIResponse(r *http.Response, v interface{}) error {
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return &ClientResponseError{body: string(b), statusCode: r.StatusCode}
	}
	if err := json.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}

func jsonFrom(jobPath string) ([]byte, error) {
	f, err := os.Open(jobPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p, err := jobspec.Parse(f)
	if err != nil {
		return nil, err
	}
	j, err := json.Marshal(payload{p})
	if err != nil {
		return nil, err
	}
	return j, nil
}
