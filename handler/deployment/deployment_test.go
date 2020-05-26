package deployment

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/s3"
	dpnethttp "github.com/ONSdigital/dp-net/http"
	nomad "github.com/ONSdigital/dp-nomad"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

const nomadURL = "http://localhost:4646"

var (
	jobSuccess = `{"EvalID": "12345", "ID": "54321", "JobModifyIndex": 99}`

	otherDeployment      = `{"JobSpecModifyIndex": 1, "ID": "54321", "Status": "failed"}`
	yetAnotherDeployment = `{"JobSpecModifyIndex": 2, "ID": "54321", "Status": "failed"}`

	deploymentSuccess = `[` + otherDeployment + `,{"JobSpecModifyIndex": 99, "Status": "successful", "StatusDescription": "Deployment completed successfully"},` + yetAnotherDeployment + `]`
	deploymentError   = `[` + otherDeployment + `,{"JobSpecModifyIndex": 99, "ID": "54321", "Status": "failed"},` + yetAnotherDeployment + `]`
	deploymentRunning = `[` + otherDeployment + `,{"JobSpecModifyIndex": 99, "ID": "54321", "Status": "running"},` + yetAnotherDeployment + `]`

	planErrors   = `{"FailedTGAllocs": { "test": {} } }`
	planSuccess  = `{}`
	planWarnings = `{"Warnings": "test warning"}`

	nomadClient = &nomad.Client{}

	normalTimeout = time.Second * 10
	shortTimeout  = time.Second * 2
)

func TestNew(t *testing.T) {
	os.Clearenv()
	os.Setenv("AWS_CREDENTIAL_FILE", "/i/hope/this/path/does/not/exist")
	defer os.Unsetenv("AWS_CREDENTIAL_FILE")

	ctx := context.Background()

	withEnv(func() {
		Convey("a deployment is returned", t, func() {
			d := New(ctx, &config.Configuration{DeploymentRoot: "foo", NomadEndpoint: "https://", NomadToken: "baz", NomadCACert: "", NomadTLSSkipVerify: false, AWSRegion: "qux"}, &s3.ClientMock{}, nomadClient)
			So(d, ShouldNotBeNil)
		})
	})
}

func TestPlan(t *testing.T) {
	withMocks(func() {

		ctx := context.Background()

		Convey("plan functions as expected", t, func() {

			Convey("api errors handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/job/test/plan", httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, nomadClient: nomadClient}
				err := dep.plan(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
			})

			Convey("plan warnings handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/job/test/plan", httpmock.NewStringResponder(200, planWarnings))
				dep := &Deployment{endpoint: nomadURL, nomadClient: nomadClient}
				err := dep.plan(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "plan for tasks generated errors or warnings")
			})

			Convey("plan allocation errors handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/job/test/plan", httpmock.NewStringResponder(200, planErrors))
				dep := &Deployment{endpoint: nomadURL, nomadClient: nomadClient}
				err := dep.plan(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "plan for tasks generated errors or warnings")
			})

			Convey("valid plans handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/job/test/plan", httpmock.NewStringResponder(200, planSuccess))
				dep := &Deployment{endpoint: nomadURL, nomadClient: nomadClient}
				err := dep.plan(ctx, &engine.Message{Service: "test"})
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestRun(t *testing.T) {
	withMocks(func() {
		Convey("run functions as expected", t, func() {

			ctx, cancel := context.WithCancel(context.Background())

			Convey("job api errors handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/jobs", httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("deployment api errors handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", nomadURL+"/v1/job/test/deployments", httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("deployment failures handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", nomadURL+"/v1/job/test/deployments", httpmock.NewStringResponder(200, deploymentError))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
				cancel()
			})

			Convey("deployment timeouts handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", nomadURL+"/v1/job/test/deployments", httpmock.NewStringResponder(200, deploymentRunning))
				dep := &Deployment{endpoint: nomadURL, timeout: shortTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete")
				cancel()
			})

			Convey("deployment cancellation handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", nomadURL+"/v1/job/test/deployments", httpmock.NewStringResponder(200, deploymentRunning))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("successful deployments handled correctly", func() {
				httpmock.RegisterResponder("POST", nomadURL+"/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", nomadURL+"/v1/job/test/deployments", httpmock.NewStringResponder(200, deploymentSuccess))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldBeNil)
				cancel()
			})
		})
	})
}

func withEnv(f func()) {
	defer os.Clearenv()
	f()
}

func withMocks(f func()) {

	// Activate() changes http.DefaultClient (see later)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	myClient := dpnethttp.DefaultClient
	// use the httpmock'ed http.DefaultClient in our dp-net http client
	myClient.HTTPClient = http.DefaultClient
	myClient.MaxRetries = 1

	nomadClient = &nomad.Client{
		Client: myClient,
		URL:    nomadURL,
	}

	defaultJSONFrom := jsonFrom

	defer func() {
		jsonFrom = defaultJSONFrom
	}()

	jsonFrom = func(string) ([]byte, error) { return nil, nil }
	f()
}
