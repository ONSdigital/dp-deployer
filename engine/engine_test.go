package engine

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/LloydGriffiths/ssqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

type mockConsumer struct {
	exhausted bool
	sqsiface.SQSAPI
	errorable bool
	message   *sqs.Message
	mu        sync.Mutex
}

func (m *mockConsumer) ReceiveMessage(in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	defer m.mu.Unlock()
	m.mu.Lock()

	if m.exhausted {
		return &sqs.ReceiveMessageOutput{Messages: nil}, nil
	}
	m.exhausted = true

	if m.errorable {
		return nil, errors.New("test consume error")
	}
	return &sqs.ReceiveMessageOutput{Messages: []*sqs.Message{m.message}}, nil
}

func (m *mockConsumer) DeleteMessage(in *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return nil, nil
}

type mockProducer struct {
	message string
}

func (m *mockProducer) SendMessage(body string) error {
	m.message = body
	return nil
}

var (
	defaultErrHandler = ErrHandler
	invalidMessage    = &sqs.Message{Body: aws.String(""), MessageId: aws.String("100"), ReceiptHandle: aws.String("")}
	validMessage      = &sqs.Message{Body: aws.String(`{"type": "test"}`), MessageId: aws.String("200"), ReceiptHandle: aws.String("")}
)

func TestNew(t *testing.T) {
	if testing.Short() {
		t.Skip("short test run - skipping")
	}

	tests := []struct {
		input    *Config
		expected string
	}{
		{&Config{"", "foo", "bar", "baz"}, "missing consumer queue name"},
		{&Config{"foo", "", "bar", "baz"}, "missing consumer queue url"},
		{&Config{"foo", "bar", "", "baz"}, "missing producer queue name"},
		{&Config{"foo", "bar", "baz", ""}, "missing queue region"},
		{&Config{"foo", "bar", "baz", "qux"}, "No valid AWS authentication found"},
	}

	for _, test := range tests {
		Convey("an error is returned with misconfiguration", t, func() {
			e, err := New(test.input, nil)
			So(e, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, test.expected)
		})
	}

	withEnv(func() {
		Convey("an engine is returned with good configuration", t, func() {
			e, err := New(&Config{"foo", "bar", "baz", "qux"}, nil)
			So(err, ShouldBeNil)
			So(e, ShouldNotBeNil)
		})
	})
}

func TestStart(t *testing.T) {
	withEnv(func() {
		Convey("start behaives correctly", t, func(c C) {
			setup := func() (*Engine, error) { return New(&Config{"foo", "bar", "baz", "qux"}, nil) }
			ctx, cancel := context.WithCancel(context.Background())

			Convey("queue errors handled correctly", func() {
				withMocks(true, invalidMessage, func(producer *mockProducer) {
					e, err := setup()
					So(err, ShouldBeNil)

					ErrHandler = func(messageID string, err error) {
						cancel()
						c.So(messageID, ShouldEqual, "")
						c.So(err.Error(), ShouldEqual, "test consume error")
					}

					e.Start(ctx)
					c.So(producer.message, ShouldEqual, "")
				})
			})

			Convey("unmarshaling errors handled correctly", func() {
				withMocks(false, invalidMessage, func(producer *mockProducer) {
					e, err := setup()
					So(err, ShouldBeNil)

					ErrHandler = func(messageID string, err error) {
						cancel()
						c.So(messageID, ShouldEqual, "100")
						c.So(err.Error(), ShouldEqual, "unexpected end of JSON input")
					}

					e.Start(ctx)
					c.So(producer.message, ShouldEqual, `{"Error":"unexpected end of JSON input","ID":"100","Success":false}`)
				})
			})

			Convey("missing handlers handled correctly", func() {
				withMocks(false, validMessage, func(producer *mockProducer) {
					e, err := setup()
					So(err, ShouldBeNil)

					ErrHandler = func(messageID string, err error) {
						cancel()
						c.So(messageID, ShouldEqual, "200")
						c.So(err.Error(), ShouldEqual, "missing handler for message type: test")
					}

					e.Start(ctx)
					c.So(producer.message, ShouldEqual, `{"Error":"missing handler for message type: test","ID":"200","Success":false}`)
				})
			})

			Convey("handler errors handled correctly", func() {
				withMocks(false, validMessage, func(producer *mockProducer) {
					e, err := setup()
					So(err, ShouldBeNil)

					hfunction := func(ctx context.Context, msg *Message) error { return errors.New("test handler error") }
					e.handlers = map[string]HandlerFunc{"test": hfunction}
					ErrHandler = func(messageID string, err error) {
						cancel()
						c.So(messageID, ShouldEqual, "200")
						c.So(err.Error(), ShouldEqual, "test handler error")
					}

					e.Start(ctx)
					So(producer.message, ShouldEqual, `{"Error":"test handler error","ID":"200","Success":false}`)
				})
			})

			Convey("successful message handles handled correctly", func() {
				withMocks(false, validMessage, func(producer *mockProducer) {
					e, err := setup()
					So(err, ShouldBeNil)

					hfunction := func(ctx context.Context, msg *Message) error { return nil }
					e.handlers = map[string]HandlerFunc{"test": hfunction}
					ErrHandler = defaultErrHandler

					go time.AfterFunc(time.Second*1, cancel)
					e.Start(ctx)
					So(producer.message, ShouldEqual, `{"ID":"200","Success":true}`)
				})
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

func withMocks(errorable bool, msg *sqs.Message, f func(*mockProducer)) {
	origClient := ssqs.DefaultClient
	origSendMsg := sendMessage

	defer func() {
		sendMessage = origSendMsg
		ssqs.DefaultClient = origClient
	}()

	ssqs.DefaultClient = func(q *ssqs.Queue) sqsiface.SQSAPI {
		return &mockConsumer{errorable: errorable, message: msg}
	}

	mockedprod := &mockProducer{}
	sendMessage = mockedprod.SendMessage
	f(mockedprod)
}
