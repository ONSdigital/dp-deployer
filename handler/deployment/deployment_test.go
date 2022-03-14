package deployment

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"

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
	jobSuccess            = `{"EvalID": "12345", "ID": "54321", "JobModifyIndex": 99}`
	serviceJobInfoSuccess = `{"ID": "54321", "Name": "test", "Type": "service", "Version": 2}`
	systemJobInfoSuccess  = `{"ID": "54321", "Name": "test", "Type": "system", "Version": 2}`
	batchJobInfoSuccess   = `{"ID": "54321", "Name": "test", "Type": "system", "Version": 2}`

	otherDeployment      = `{"JobSpecModifyIndex": 1, "ID": "54321", "Status": "failed"}`
	yetAnotherDeployment = `{"JobSpecModifyIndex": 2, "ID": "54321", "Status": "failed"}`

	deploymentSuccess = `[` + otherDeployment + `,{"JobSpecModifyIndex": 99, "Status": "successful", "StatusDescription": "Deployment completed successfully"},` + yetAnotherDeployment + `]`
	deploymentError   = `[` + otherDeployment + `,{"JobSpecModifyIndex": 99, "ID": "54321", "Status": "failed"},` + yetAnotherDeployment + `]`
	deploymentRunning = `[` + otherDeployment + `,{"JobSpecModifyIndex": 99, "ID": "54321", "Status": "running"},` + yetAnotherDeployment + `]`

	emptyAllocations         = `[]`
	anAllocation             = `{"ID": "54321", "JobVersion": 2, "ClientStatus": "running", "DesiredStatus": "run"}`
	anotherAllocation        = `{"ID": "54322", "JobVersion": 2, "ClientStatus": "running", "DesiredStatus": "run"}`
	allocationsSuccess       = `[` + anAllocation + `, ` + anotherAllocation + `]`
	allocationsPending       = `[` + anAllocation + `, {"ID": "54322", "JobVersion": 2, "ClientStatus": "pending", "DesiredStatus": "run"}]`
	allocationsOldVersion    = `[{"ID": "54321", "JobVersion": 1, "ClientStatus": "running", "DesiredStatus": "run"}, ` + anotherAllocation + `]`
	allocationsError         = `[` + anAllocation + `, {"ID": "54322", "JobVersion": 2, "ClientStatus": "failed", "DesiredStatus": "run"}]`
	allocationsStopIsRunning = `[` + anAllocation + `, {"ID": "54322", "JobVersion": 1, "ClientStatus": "running", "DesiredStatus": "stop"}]`
	allocationsStopIsStopped = `[` + anAllocation + `, {"ID": "54322", "JobVersion": 1, "ClientStatus": "complete", "DesiredStatus": "stop"}]`

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

	withEnv(func() {
		Convey("a deployment is returned", t, func() {
			d := New(&config.Configuration{DeploymentRoot: "foo", NomadEndpoint: "https://", NomadToken: "baz", NomadCACert: "", NomadTLSSkipVerify: false, AWSRegion: "qux"}, &s3.ClientMock{}, nomadClient)
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
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("job info api errors handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(500, "server error"))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("service deployment api errors handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, serviceJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, serviceName), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("service deployment api failures handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, serviceJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, serviceName), httpmock.NewStringResponder(200, deploymentError))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
				cancel()
			})

			Convey("service deployment timeouts handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, serviceJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, serviceName), httpmock.NewStringResponder(200, deploymentRunning))
				dep := &Deployment{endpoint: nomadURL, timeout: shortTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete")
				cancel()
			})

			Convey("service deployment cancellation handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, serviceJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, serviceName), httpmock.NewStringResponder(200, deploymentRunning))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("successful service deployments handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, serviceJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, serviceName), httpmock.NewStringResponder(200, deploymentSuccess))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldBeNil)
				cancel()
			})

			Convey("system allocations api errors handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("empty system allocations api response handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, emptyAllocations))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
				cancel()
			})

			Convey("system deployment timeouts handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsPending))
				dep := &Deployment{endpoint: nomadURL, timeout: shortTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete")
				cancel()
			})

			Convey("system deployment cancellation handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsPending))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("system deployment failed allocation handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsError))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("system deployment old version persists handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsOldVersion))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("system deployment allocation desired 'stop' still running handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsStopIsRunning))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("successful system deployments handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsSuccess))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldBeNil)
				cancel()
			})

			Convey("successful system deployments with allocation desired 'stop' stopped handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, systemJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsStopIsStopped))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldBeNil)
				cancel()
			})

			Convey("batch allocations api errors handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("empty batch allocations api response handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, emptyAllocations))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
				cancel()
			})

			Convey("batch deployment timeouts handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsPending))
				dep := &Deployment{endpoint: nomadURL, timeout: shortTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: serviceName})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete")
				cancel()
			})

			Convey("batch deployment cancellation handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsPending))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("batch deployment failed allocation handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsError))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("batch deployment old version persists handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsOldVersion))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("batch deployment allocation desired 'stop' still running handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsStopIsRunning))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("successful batch deployments handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsSuccess))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldBeNil)
				cancel()
			})

			Convey("successful batch deployments with allocation desired 'stop' stopped handled correctly", func() {
				serviceName := "test"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(infoURL, nomadURL, serviceName), httpmock.NewStringResponder(200, batchJobInfoSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, serviceName), httpmock.NewStringResponder(200, allocationsStopIsStopped))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.run(ctx, &engine.Message{ID: "54321", Service: "test"})
				So(err, ShouldBeNil)
				cancel()
			})
		})
	})
}

func TestRunNew(t *testing.T) {
	withMocks(func() {
		Convey("runNew functions as expected", t, func() {

			ctx, cancel := context.WithCancel(context.Background())
			jobID := "54321"
			jobName := "test"
			var jobVersion uint64 = 2

			Convey("job api errors handled correctly", func() {
				jobType := "service"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("service deployment api errors handled correctly", func() {
				jobType := "service"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, jobName), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("service deployment api failures handled correctly", func() {
				jobType := "service"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, jobName), httpmock.NewStringResponder(200, deploymentError))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
				cancel()
			})

			Convey("service deployment timeouts handled correctly", func() {
				jobType := "service"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, jobName), httpmock.NewStringResponder(200, deploymentRunning))
				dep := &Deployment{endpoint: nomadURL, timeout: shortTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete")
				cancel()
			})

			Convey("service deployment cancellation handled correctly", func() {
				jobType := "service"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, jobName), httpmock.NewStringResponder(200, deploymentRunning))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("successful service deployments handled correctly", func() {
				jobType := "service"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(deploymentURL, nomadURL, jobName), httpmock.NewStringResponder(200, deploymentSuccess))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldBeNil)
				cancel()
			})

			Convey("system allocations api errors handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("empty system allocations api response handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, emptyAllocations))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
				cancel()
			})

			Convey("system deployment timeouts handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsPending))
				dep := &Deployment{endpoint: nomadURL, timeout: shortTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete")
				cancel()
			})

			Convey("system deployment cancellation handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsPending))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("system deployment failed allocation handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsError))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("system deployment old version persists handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsOldVersion))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("system deployment allocation desired 'stop' still running handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsStopIsRunning))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("successful system deployments handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsSuccess))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldBeNil)
				cancel()
			})

			Convey("successful system deployments with allocation desired 'stop' stopped handled correctly", func() {
				jobType := "system"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsStopIsStopped))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldBeNil)
				cancel()
			})

			Convey("batch allocations api errors handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client")
				cancel()
			})

			Convey("empty batch allocations api response handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, emptyAllocations))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
				cancel()
			})

			Convey("batch deployment timeouts handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsPending))
				dep := &Deployment{endpoint: nomadURL, timeout: shortTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete")
				cancel()
			})

			Convey("batch deployment cancellation handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsPending))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("batch deployment failed allocation handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsError))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("batch deployment old version persists handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsOldVersion))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("batch deployment allocation desired 'stop' still running handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsStopIsRunning))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring deployment")
			})

			Convey("successful batch deployments handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsSuccess))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
				So(err, ShouldBeNil)
				cancel()
			})

			Convey("successful batch deployments with allocation desired 'stop' stopped handled correctly", func() {
				jobType := "batch"
				httpmock.RegisterResponder("POST", fmt.Sprintf(runURL, nomadURL), httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", fmt.Sprintf(allocationsURL, nomadURL, jobName), httpmock.NewStringResponder(200, allocationsStopIsStopped))
				dep := &Deployment{endpoint: nomadURL, timeout: normalTimeout, nomadClient: nomadClient}
				err := dep.runNew(ctx, api.Job{ID: &jobID, Name: &jobName, Type: &jobType, Version: &jobVersion})
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

func TestPatchJob(t *testing.T) {
	Convey("run functions as expected", t, func() {
		Convey("Given a job with no patches needed, then no changes are seen", func() {
			f := strings.NewReader(nomadJobNoPatchNeeded)
			p, err := jobspec.Parse(f)
			So(err, ShouldBeNil)

			patchJob(p)

			fWant := strings.NewReader(nomadJobNoPatchNeeded)
			pWant, err := jobspec.Parse(fWant)
			// check nothing in the job jas been changed
			So(reflect.DeepEqual(p, pWant), ShouldBeTrue)
		})

		Convey("Given a job with patches needed, then expected changes are seen", func() {
			f := strings.NewReader(nomadJobPatchNeeded)
			p, err := jobspec.Parse(f)
			So(err, ShouldBeNil)

			patchJob(p)

			So(*p.TaskGroups[0].Name, ShouldEqual, wPatchTo)
			So(p.TaskGroups[0].Constraints[0].RTarget, ShouldEqual, wPatchTo)
			So(p.TaskGroups[0].Tasks[0].Services[0].Tags[0], ShouldEqual, wPatchTo)

			So(*p.TaskGroups[1].Name, ShouldEqual, pPatchTo)
			So(p.TaskGroups[1].Constraints[0].RTarget, ShouldEqual, pPatchTo)
			So(p.TaskGroups[1].Tasks[0].Services[0].Tags[0], ShouldEqual, pPatchTo)
		})
	})
}

var nomadJobNoPatchNeeded string = `job "dp-cantabular-api-ext" {
	datacenters = ["eu-west-1"]
	region      = "eu"
	type        = "service"
	
	update {
	  stagger          = "60s"
	  min_healthy_time = "30s"
	  healthy_deadline = "2m"
	  max_parallel     = 1
	  auto_revert      = true
	}
  
	group "web" {
	  count = "1"
  
	  constraint {
		attribute = "${node.class}"
		value     = "web"
	  }
  
	  restart {
		attempts = 3
		delay    = "15s"
		interval = "1m"
		mode     = "delay"
	  }
  
	  task "dp-cantabular-api-ext-web" {
		driver = "docker"
  
		artifact {
		  source = "s3::https://s3-eu-west-1.amazonaws.com/{{DEPLOYMENT_BUCKET}}/dp-cantabular-api-ext/{{TARGET_ENVIRONMENT}}/{{RELEASE}}.tar.gz"
		}
  
		config {
		  command = "${NOMAD_TASK_DIR}/start-task"
  
		  args = ["./dp-cantabular-api-ext"]
  
		  image = "{{ECR_URL}}:concourse-{{REVISION}}"
  
		  port_map {
			http = "${NOMAD_PORT_http}"
		  }
		}
  
		service {
		  name = "dp-cantabular-api-ext"
		  port = "http"
		  tags = ["web"]
		}
  
		resources {
		  cpu    = "500"
		  memory = "1000"
  
		  network {
			port "http" {}
		  }
		}
  
		template {
		  source      = "${NOMAD_TASK_DIR}/vars-template"
		  destination = "${NOMAD_TASK_DIR}/vars"
		}
  
		vault {
		  policies = ["dp-cantabular-api-ext-web"]
		}
	  }
	}
  
	group "publishing" {
	  count = "1"
  
	  constraint {
		attribute = "${node.class}"
		value     = "publishing"
	  }
  
	  restart {
		attempts = 3
		delay    = "15s"
		interval = "1m"
		mode     = "delay"
	  }
  
	  task "dp-cantabular-api-ext-publishing" {
		driver = "docker"
  
		artifact {
		  source = "s3::https://s3-eu-west-1.amazonaws.com/{{DEPLOYMENT_BUCKET}}/dp-cantabular-api-ext/{{TARGET_ENVIRONMENT}}/{{RELEASE}}.tar.gz"
		}
  
		config {
		  command = "${NOMAD_TASK_DIR}/start-task"
  
		  args = ["./dp-cantabular-api-ext"]
  
		  image = "{{ECR_URL}}:concourse-{{REVISION}}"
  
		  port_map {
			http = "${NOMAD_PORT_http}"
		  }
		}
  
		service {
		  name = "dp-cantabular-api-ext"
		  port = "http"
		  tags = ["publishing"]
		}
  
		resources {
		  cpu    = "500"
		  memory = "1000"
  
		  network {
			port "http" {}
		  }
		}
  
		template {
		  source      = "${NOMAD_TASK_DIR}/vars-template"
		  destination = "${NOMAD_TASK_DIR}/vars"
		}
  
		vault {
		  policies = ["dp-cantabular-api-ext-publishing"]
		}
	  }
	}
  }`

// the following contains fields of 'web_cantabular' or 'publishing_cantabular' for patching
var nomadJobPatchNeeded string = `job "dp-cantabular-api-ext" {
	datacenters = ["eu-west-1"]
	region      = "eu"
	type        = "service"
	
	update {
	  stagger          = "60s"
	  min_healthy_time = "30s"
	  healthy_deadline = "2m"
	  max_parallel     = 1
	  auto_revert      = true
	}
  
	group "web_cantabular" {
	  count = "1"
  
	  constraint {
		attribute = "${node.class}"
		value     = "web_cantabular"
	  }
  
	  restart {
		attempts = 3
		delay    = "15s"
		interval = "1m"
		mode     = "delay"
	  }
  
	  task "dp-cantabular-api-ext-web" {
		driver = "docker"
  
		artifact {
		  source = "s3::https://s3-eu-west-1.amazonaws.com/{{DEPLOYMENT_BUCKET}}/dp-cantabular-api-ext/{{TARGET_ENVIRONMENT}}/{{RELEASE}}.tar.gz"
		}
  
		config {
		  command = "${NOMAD_TASK_DIR}/start-task"
  
		  args = ["./dp-cantabular-api-ext"]
  
		  image = "{{ECR_URL}}:concourse-{{REVISION}}"
  
		  port_map {
			http = "${NOMAD_PORT_http}"
		  }
		}
  
		service {
		  name = "dp-cantabular-api-ext"
		  port = "http"
		  tags = ["web_cantabular"]
		}
  
		resources {
		  cpu    = "500"
		  memory = "1000"
  
		  network {
			port "http" {}
		  }
		}
  
		template {
		  source      = "${NOMAD_TASK_DIR}/vars-template"
		  destination = "${NOMAD_TASK_DIR}/vars"
		}
  
		vault {
		  policies = ["dp-cantabular-api-ext-web"]
		}
	  }
	}
  
	group "publishing_cantabular" {
	  count = "1"
  
	  constraint {
		attribute = "${node.class}"
		value     = "publishing_cantabular"
	  }
  
	  restart {
		attempts = 3
		delay    = "15s"
		interval = "1m"
		mode     = "delay"
	  }
  
	  task "dp-cantabular-api-ext-publishing" {
		driver = "docker"
  
		artifact {
		  source = "s3::https://s3-eu-west-1.amazonaws.com/{{DEPLOYMENT_BUCKET}}/dp-cantabular-api-ext/{{TARGET_ENVIRONMENT}}/{{RELEASE}}.tar.gz"
		}
  
		config {
		  command = "${NOMAD_TASK_DIR}/start-task"
  
		  args = ["./dp-cantabular-api-ext"]
  
		  image = "{{ECR_URL}}:concourse-{{REVISION}}"
  
		  port_map {
			http = "${NOMAD_PORT_http}"
		  }
		}
  
		service {
		  name = "dp-cantabular-api-ext"
		  port = "http"
		  tags = ["publishing_cantabular"]
		}
  
		resources {
		  cpu    = "500"
		  memory = "1000"
  
		  network {
			port "http" {}
		  }
		}
  
		template {
		  source      = "${NOMAD_TASK_DIR}/vars-template"
		  destination = "${NOMAD_TASK_DIR}/vars"
		}
  
		vault {
		  policies = ["dp-cantabular-api-ext-publishing"]
		}
	  }
	}
  }`
