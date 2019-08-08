// Package deployment provides functionality for planning and running deployment jobs.
package deployment

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ONSdigital/dp-deployer/engine"
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

var jsonFrom func(string) ([]byte, error)

type payload struct {
	Job *api.Job
}

// HTTPClient is the default http client.
var HTTPClient = &http.Client{Timeout: time.Second * 10}

// Config represents the configuration for a deployment.
type Config struct {
	// DeploymentRoot is the path to root of deployments.
	DeploymentRoot string
	// NomadEndpoint is the Nomad client endpoint.
	NomadEndpoint string
	// NomadToken is the ACL token used to authorise HTTP requests.
	NomadToken string
	// NomadCACert is the path to the root nomad CA cert.
	NomadCACert string
	// Region is the region in which the deployment artifacts bucket resides.
	Region string
	// Timeout is the timeout configuration for the deployments.
	Timeout *TimeoutConfig
}

// TimeoutConfig represents the configuration for deployment timeouts.
type TimeoutConfig struct {
	// Allocation is the max time to wait for all allocations to complete.
	Allocation time.Duration
	// Evaluation is the max time to wait for an Evaluation to complete.
	Evaluation time.Duration
}

// Deployment represents a deployment.
type Deployment struct {
	s3Client    *s3.S3
	nomadClient *http.Client
	root        string
	endpoint    string
	token       string
	timeout     *TimeoutConfig
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

	NomadClient := HTTPClient
	if c.NomadCACert != "" {
		log.Trace("loading custom ca cert", log.Data{"ca_cert_path": c.NomadCACert})

		caCertPool, _ := x509.SystemCertPool()
		if caCertPool == nil {
			caCertPool = x509.NewCertPool()
		}

		caCert, err := ioutil.ReadFile(c.NomadCACert)
		if err != nil {
			return nil, err
		}
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to append ca cert to pool")
		}

		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		NomadClient = &http.Client{
			Timeout:   time.Second * 10,
			Transport: transport,
		}
	} else if strings.HasPrefix(c.NomadEndpoint, "https://localhost:") {
		// no CA file, using local nomad => do not check cert  XXX DANGER DANGER XXX
		log.Trace("using TLS without verification", nil)
		NomadClient.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
	}

	if jsonFrom == nil {
		jsonFrom = jsonFromFile
	}

	return &Deployment{
		s3Client:    s3.New(a, aws.Regions[c.Region], HTTPClient),
		nomadClient: NomadClient,
		root:        c.DeploymentRoot,
		endpoint:    c.NomadEndpoint,
		token:       c.NomadToken,
		timeout:     c.Timeout,
	}, nil
}

// Handler handles deployment messages that are delegated by the engine.
func (d *Deployment) Handler(ctx context.Context, msg *engine.Message) error {
	b, err := d.s3Client.Bucket(msg.Bucket).Get(msg.Artifacts[0])
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
	log.TraceC(msg.ID, "planning job", log.Data{"msg": msg, "service": msg.Service})

	var res api.JobPlanResponse
	if err := d.post(fmt.Sprintf(planURL, d.endpoint, msg.Service), msg, &res); err != nil {
		return err
	}
	if len(res.Warnings) == 0 && res.FailedTGAllocs == nil {
		return nil
	}
	if len(res.Warnings) > 0 {
		return &PlanError{Service: msg.Service, Warnings: res.Warnings}
	}
	j, err := json.Marshal(res.FailedTGAllocs)
	if err != nil {
		return err
	}
	return &PlanError{Errors: string(j), Service: msg.Service}
}

func (d *Deployment) run(ctx context.Context, msg *engine.Message) error {
	log.TraceC(msg.ID, "running job", log.Data{"msg": msg, "service": msg.Service})

	var res api.JobRegisterResponse
	if err := d.post(fmt.Sprintf(runURL, d.endpoint), msg, &res); err != nil {
		return err
	}
	if err := d.monitor(ctx, msg.ID, res.EvalID); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) monitor(ctx context.Context, deploymentID, evaluationID string) error {
	if err := d.evaluation(ctx, deploymentID, evaluationID); err != nil {
		return err
	}
	if err := d.allocations(ctx, deploymentID, evaluationID); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) allocations(ctx context.Context, deploymentID, evaluationID string) error {
	ticker := time.Tick(time.Second * 1)
	timeout := time.After(d.timeout.Allocation)

	for {
		select {
		case <-ctx.Done():
			log.InfoC(deploymentID, "bailing on deployment allocations", log.Data{"evaluation": evaluationID})
			return &AllocationAbortedError{EvaluationID: evaluationID}
		case <-timeout:
			return &TimeoutError{Action: "allocation"}
		case <-ticker:
			var allocations []api.Allocation
			if err := d.get(fmt.Sprintf(allocURL, d.endpoint, evaluationID), &allocations); err != nil {
				return err
			}
			pending, running := sumAllocations(&allocations)
			if pending > 0 {
				log.TraceC(deploymentID, "allocations still pending", log.Data{"evaluation": evaluationID, "pending": pending, "total": len(allocations)})
				continue
			}
			if running == len(allocations) {
				log.TraceC(deploymentID, "all allocations running", log.Data{"evaluation": evaluationID, "running": running, "total": len(allocations)})
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

func (d *Deployment) evaluation(ctx context.Context, deploymentID, evaluationID string) error {
	ticker := time.Tick(time.Second * 1)
	timeout := time.After(d.timeout.Evaluation)

	for {
		select {
		case <-ctx.Done():
			log.InfoC(deploymentID, "bailing on deployment evaluation", log.Data{"evaluation": evaluationID})
			return &EvaluationAbortedError{ID: evaluationID}
		case <-timeout:
			return &TimeoutError{Action: "evaluation"}
		case <-ticker:
			var evaluation api.Evaluation
			if err := d.get(fmt.Sprintf(evalURL, d.endpoint, evaluationID), &evaluation); err != nil {
				return err
			}
			if evaluation.Status == statusPending {
				log.TraceC(deploymentID, "waiting for evaluation to be scheduled", log.Data{"id": evaluation.ID})
				continue
			}
			if evaluation.Status != statusComplete {
				return &EvaluationError{ID: evaluation.ID}
			}
			log.TraceC(deploymentID, "evaluation complete", log.Data{"id": evaluation.ID})
			if len(evaluation.NextEval) == 0 {
				return nil
			}
			log.InfoC(deploymentID, "waiting for next evaluation", log.Data{"id": evaluation.ID, "next evaluation": evaluation.NextEval})
			return d.monitor(ctx, deploymentID, evaluation.NextEval)
		default:
		}
	}
}

func (d *Deployment) get(url string, v interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nomad-Token", d.token)

	res, err := d.nomadClient.Do(req)
	if err != nil {
		return err
	}
	return unmarshalAPIResponse(res, v)
}

func (d *Deployment) post(url string, msg *engine.Message, v interface{}) error {
	j, err := jsonFrom(fmt.Sprintf("%s/%s/%s.nomad", d.root, msg.Service, msg.Service))
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(j))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nomad-Token", d.token)

	res, err := d.nomadClient.Do(req)
	if err != nil {
		return err
	}
	return unmarshalAPIResponse(res, v)
}

func unmarshalAPIResponse(r *http.Response, v interface{}) error {
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return &ClientResponseError{Body: string(b), StatusCode: r.StatusCode}
	}
	if err := json.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}

func jsonFromFile(jobPath string) ([]byte, error) {
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
