package deployment

import (
	"context"
	"os"
	"testing"
	"time"

	httpmock "gopkg.in/jarcoal/httpmock.v1"

	"github.com/ONSdigital/dp-ci/awdry/engine"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	allocationError    = `[{"ClientStatus": "failed"}]`
	allocationPending  = `[{"ClientStatus": "pending"}]`
	allocationRunning  = `[{"ClientStatus": "running"}]`
	evaluationComplete = `{"Status": "complete"}`
	evaluationError    = `{"ID": "12345", "Status": "failed"}`
	evaluationPending  = `{"ID": "12345", "Status": "pending"}`
	jobSuccess         = `{"EvalID": "12345"}`
	planErrors         = `{"FailedTGAllocs": { "test": {} } }`
	planSuccess        = `{}`
	planWarnings       = `{"Warnings": "test warning"}`
)

func TestNew(t *testing.T) {
	if testing.Short() {
		t.Skip("short test run - skipping")
	}

	Convey("an error is returned with misconfiguration", t, func() {
		d, err := New(&Config{"foo", "bar", "baz", nil})
		So(d, ShouldBeNil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "No valid AWS authentication found")
	})

	withEnv(func() {
		Convey("timeout configuration defaults are used when timeout is nil", t, func() {
			d, err := New(&Config{"foo", "bar", "baz", nil})
			So(err, ShouldBeNil)
			So(d, ShouldNotBeNil)
			So(d.timeout.Allocation, ShouldEqual, DefaultAllocationTimeout)
			So(d.timeout.Evaluation, ShouldEqual, DefaultEvaluationTimeout)
		})
	})

	withEnv(func() {
		Convey("timeout configuration defaults are used when timeout is unreasonable", t, func() {
			d, err := New(&Config{"foo", "bar", "baz", &TimeoutConfig{0, 0}})
			So(err, ShouldBeNil)
			So(d, ShouldNotBeNil)
			So(d.timeout.Allocation, ShouldEqual, DefaultAllocationTimeout)
			So(d.timeout.Evaluation, ShouldEqual, DefaultEvaluationTimeout)
		})
	})
}

func TestPlan(t *testing.T) {
	withMocks(func() {
		Convey("plan behaives correctly", t, func() {
			httpmock.DeactivateAndReset()
			httpmock.Activate()

			Convey("api errors handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/job/test/plan", httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: "http://localhost:4646"}
				err := dep.plan(&engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client (500): server error")
			})

			Convey("plan warnings handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/job/test/plan", httpmock.NewStringResponder(200, planWarnings))
				dep := &Deployment{endpoint: "http://localhost:4646"}
				err := dep.plan(&engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "plan for service test generated errors or warnings")
			})

			Convey("plan allocation errors handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/job/test/plan", httpmock.NewStringResponder(200, planErrors))
				dep := &Deployment{endpoint: "http://localhost:4646"}
				err := dep.plan(&engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "plan for service test generated errors or warnings")
			})

			Convey("valid plans handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/job/test/plan", httpmock.NewStringResponder(200, planSuccess))
				dep := &Deployment{endpoint: "http://localhost:4646"}
				err := dep.plan(&engine.Message{Service: "test"})
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestRun(t *testing.T) {
	withMocks(func() {
		Convey("run behaives correctly", t, func() {
			httpmock.DeactivateAndReset()
			httpmock.Activate()

			ctx, cancel := context.WithCancel(context.Background())

			Convey("job api errors handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: "http://localhost:4646"}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client (500): server error")
				cancel()
			})

			Convey("evaluation api errors handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(403, "client error"))
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 10, time.Second * 10}}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client (403): client error")
				cancel()
			})

			Convey("evaluation failures handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(200, evaluationError))
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 10, time.Second * 10}}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "error occurred for evaluation: 12345")
				cancel()
			})

			Convey("evaluation timeouts handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(200, evaluationPending))
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 2, time.Second * 2}}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete: evaluation")
				cancel()
			})

			Convey("evaluation cancellation handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(200, evaluationPending))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 10, time.Second * 10}}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring evaluation: 12345")
			})

			Convey("allocation api errors handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(200, evaluationComplete))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345/allocations", httpmock.NewStringResponder(500, "server error"))
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 10, time.Second * 10}}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unexpected response from client (500): server error")
				cancel()
			})

			Convey("allocation failures handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(200, evaluationComplete))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345/allocations", httpmock.NewStringResponder(200, allocationError))
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 10, time.Second * 10}}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "failed to start all allocations (pending: 0, running: 0, total: 1)")
				cancel()
			})

			Convey("allocation timeouts handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(200, evaluationComplete))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345/allocations", httpmock.NewStringResponder(200, allocationPending))
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 2, time.Second * 2}}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "timed out waiting for action to complete: allocation")
				cancel()
			})

			Convey("allocation cancellation handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(200, evaluationComplete))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345/allocations", httpmock.NewStringResponder(200, allocationPending))
				time.AfterFunc(time.Second*2, cancel)
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 10, time.Second * 10}}
				err := dep.run(ctx, &engine.Message{Service: "test"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "aborted monitoring allocations for evaluation 12345")
			})

			Convey("successful allocations handled correctly", func() {
				httpmock.RegisterResponder("POST", "http://localhost:4646/v1/jobs", httpmock.NewStringResponder(200, jobSuccess))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345", httpmock.NewStringResponder(200, evaluationComplete))
				httpmock.RegisterResponder("GET", "http://localhost:4646/v1/evaluation/12345/allocations", httpmock.NewStringResponder(200, allocationRunning))
				dep := &Deployment{endpoint: "http://localhost:4646", timeout: &TimeoutConfig{time.Second * 10, time.Second * 10}}
				err := dep.run(ctx, &engine.Message{ID: "test", Service: "test"})
				So(err, ShouldBeNil)
				cancel()
			})
		})
	})
}

func withEnv(f func()) {
	defer os.Clearenv()

	os.Clearenv()
	os.Setenv("AWS_ACCESS_KEY_ID", "FOO")
	os.Setenv("AWS_DEFAULT_REGION", "BAR")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "BAZ")

	f()
}

func withMocks(f func()) {
	origJSONFrom := jsonFrom

	defer func() {
		jsonFrom = origJSONFrom
	}()

	jsonFrom = func(string) ([]byte, error) { return nil, nil }
	f()
}
