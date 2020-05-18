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

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/s3"
	nomad "github.com/ONSdigital/dp-nomad"
	"github.com/ONSdigital/log.go/log"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/slimsag/untargz"
)

const (
	deploymentURL = "%s/v1/job/%s/deployments"
	planURL       = "%s/v1/job/%s/plan"
	runURL        = "%s/v1/jobs"

	statusComplete = "complete"
	statusPending  = "pending"
	statusRunning  = "running"
)

var jsonFrom func(string) ([]byte, error)

type payload struct {
	Job *api.Job
}

// HTTPClient is the default http client - used for s3
var HTTPClient = &http.Client{Timeout: time.Second * 10}

// Deployment represents a deployment.
type Deployment struct {
	s3Client    s3.Client
	nomadClient *nomad.Nomad
	root        string
	endpoint    string
	token       string
	timeout     time.Duration
}

// New returns a new deployment.
func New(ctx context.Context, cfg *config.Configuration, deploymentsClient s3.Client, nomadClient *nomad.Nomad) (*Deployment, error) {

	// NomadClient := HTTPClient
	// if strings.HasPrefix(cfg.NomadEndpoint, "https://") {
	// 	var tlsConfig *tls.Config
	// 	if cfg.NomadCACert != "" {
	// 		log.Event(ctx, "loading custom ca cert", log.INFO, log.Data{"ca_cert_path": cfg.NomadCACert})

	// 		caCertPool, _ := x509.SystemCertPool()
	// 		if caCertPool == nil {
	// 			caCertPool = x509.NewCertPool()
	// 		}

	// 		caCert, err := ioutil.ReadFile(cfg.NomadCACert)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		if !caCertPool.AppendCertsFromPEM(caCert) {
	// 			return nil, errors.New("failed to append ca cert to pool")
	// 		}

	// 		tlsConfig = &tls.Config{
	// 			RootCAs: caCertPool,
	// 		}
	// 	} else if cfg.NomadTLSSkipVerify {

	// 		// no CA file => do not check cert  XXX DANGER DANGER XXX
	// 		log.Event(ctx, "using TLS without verification", log.WARN)
	// 		tlsConfig = &tls.Config{
	// 			InsecureSkipVerify: true,
	// 		}
	// 	} else {
	// 		return nil, errors.New("invalid configuration with https but no CA cert or skip verification enabled")
	// 	}
	// 	NomadClient.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	// }

	if jsonFrom == nil {
		jsonFrom = jsonFromFile
	}

	return &Deployment{
		s3Client:    deploymentsClient,
		nomadClient: nomadClient,
		root:        cfg.DeploymentRoot,
		endpoint:    cfg.NomadEndpoint,
		token:       cfg.NomadToken,
		timeout:     cfg.DeploymentTimeout,
	}, nil
}

// Handler handles deployment messages that are delegated by the engine.
func (d *Deployment) Handler(ctx context.Context, msg *engine.Message) error {
	b, _, err := d.s3Client.Get(msg.Artifacts[0])
	if err != nil {
		return err
	}
	if err := untargz.Extract(b, fmt.Sprintf("%s/%s", d.root, msg.Service), nil); err != nil {
		return err
	}
	if err := d.plan(ctx, msg); err != nil {
		return err
	}
	if err := d.run(ctx, msg); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) plan(ctx context.Context, msg *engine.Message) error {
	log.Event(ctx, "planning job", log.INFO, log.Data{"msg": msg, "service": msg.Service})

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
	log.Event(ctx, "running job", log.INFO, log.Data{"msg": msg, "service": msg.Service})

	var res api.JobRegisterResponse
	if err := d.post(fmt.Sprintf(runURL, d.endpoint), msg, &res); err != nil {
		return err
	}
	if err := d.deploymentSuccess(ctx, msg.ID, res.EvalID, msg.Service, res.JobModifyIndex); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) deploymentSuccess(ctx context.Context, correlationID, evaluationID, jobID string, jobSpecModifyIndex uint64) error {
	ticker := time.Tick(time.Second * 1)
	timeout := time.After(d.timeout)
	minLogData := log.Data{"evaluation": evaluationID, "job": jobID, "job_modify_index": jobSpecModifyIndex}

	for {
		select {
		case <-ctx.Done():
			log.Event(ctx, "bailing on deployment status", log.ERROR, minLogData)
			return &AbortedError{EvaluationID: evaluationID, CorrelationID: correlationID}
		case <-timeout:
			return &TimeoutError{Action: "deployment"}
		case <-ticker:
			var deployments []api.Deployment
			if err := d.get(fmt.Sprintf(deploymentURL, d.endpoint, jobID), &deployments); err != nil {
				return err
			}
			foundJobByIndex := false
			for _, deployment := range deployments {
				if deployment.JobSpecModifyIndex != jobSpecModifyIndex {
					continue
				}

				logData := log.Data{
					"evaluation":          evaluationID,
					"job":                 deployment.JobID,
					"job_spec_modify_idx": jobSpecModifyIndex,
					"status":              deployment.Status,
					"status_desc":         deployment.StatusDescription,
				}

				switch deployment.Status {
				case structs.DeploymentStatusSuccessful:
					log.Event(ctx, "deployment success", log.INFO, logData)
					return nil
				case structs.DeploymentStatusFailed,
					structs.DeploymentStatusCancelled:

					log.Event(ctx, "deployment failed", log.ERROR, logData)
					return &AbortedError{EvaluationID: evaluationID, CorrelationID: correlationID}
				}
				foundJobByIndex = true
				break
			}
			if foundJobByIndex {
				log.Event(ctx, "deployment incomplete - will re-test", log.WARN, minLogData)
			} else {
				log.Event(ctx, "deployment not found - will re-test", log.WARN, minLogData)
			}
		}
	}
}

func (d *Deployment) get(url string, v interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	return d.doNomadReq(req, v)
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
	return d.doNomadReq(req, v)
}

func (d *Deployment) doNomadReq(req *http.Request, v interface{}) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nomad-Token", d.token)

	res, err := d.nomadClient.Client.Do(context.Background(), req)
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
		return &ClientResponseError{Body: string(b), StatusCode: r.StatusCode, URL: r.Request.URL.String()}
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
