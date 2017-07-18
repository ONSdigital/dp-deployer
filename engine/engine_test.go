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
}

var mu sync.Mutex

func (m *mockConsumer) ReceiveMessage(in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	defer mu.Unlock()
	mu.Lock()

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
	invalidMessage = &sqs.Message{Body: aws.String(""), MessageId: aws.String(""), ReceiptHandle: aws.String("")}
	validMessage   = &sqs.Message{Body: aws.String(`{"type": "test"}`), MessageId: aws.String(""), ReceiptHandle: aws.String("")}
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
	config := &Config{"foo", "bar", "baz", "qux"}

	withEnv(withMocks(true, invalidMessage, func(producer *mockProducer) {
		Convey("queue errors handled correctly", t, func(c C) {
			e, err := New(config, nil)
			c.So(err, ShouldBeNil)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ErrHandler = func(err error) {
				cancel()
				c.So(err.Error(), ShouldEqual, "test consume error")
			}

			e.Start(ctx)
			c.So(producer.message, ShouldEqual, "")
		})
	}))

	withEnv(withMocks(false, invalidMessage, func(producer *mockProducer) {
		Convey("unmarshaling errors handled correctly", t, func(c C) {
			e, err := New(config, nil)
			c.So(err, ShouldBeNil)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ErrHandler = func(err error) {
				cancel()
				c.So(err.Error(), ShouldEqual, "unexpected end of JSON input")
			}

			e.Start(ctx)
			c.So(producer.message, ShouldEqual, `{"Error":"unexpected end of JSON input","ID":"","Success":false}`)
		})
	}))

	withEnv(withMocks(false, validMessage, func(producer *mockProducer) {
		Convey("missing handlers handled correctly", t, func(c C) {
			e, err := New(config, nil)
			c.So(err, ShouldBeNil)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ErrHandler = func(err error) {
				cancel()
				c.So(err.Error(), ShouldEqual, "missing handler for message type: test")
			}

			e.Start(ctx)
			c.So(producer.message, ShouldEqual, `{"Error":"missing handler for message type: test","ID":"","Success":false}`)
		})
	}))

	withEnv(withMocks(false, validMessage, func(producer *mockProducer) {
		Convey("handler errors handled correctly", t, func(c C) {
			fn := func(ctx context.Context, msg *Message) error { return errors.New("test handler error") }
			hs := map[string]HandlerFunc{"test": fn}

			e, err := New(config, hs)
			So(err, ShouldBeNil)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ErrHandler = func(err error) {
				cancel()
				c.So(err.Error(), ShouldEqual, "test handler error")
			}

			e.Start(ctx)
			So(producer.message, ShouldEqual, `{"Error":"test handler error","ID":"","Success":false}`)
		})
	}))

	withEnv(withMocks(false, validMessage, func(producer *mockProducer) {
		Convey("successful message handles handled correctly", t, func() {
			fn := func(ctx context.Context, msg *Message) error { return nil }
			hs := map[string]HandlerFunc{"test": fn}

			e, err := New(config, hs)
			So(err, ShouldBeNil)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				for producer.message == "" {
					time.Sleep(time.Millisecond * 100)
				}
				cancel()
			}()

			e.Start(ctx)
			So(producer.message, ShouldEqual, `{"ID":"","Success":true}`)
		})
	}))
}

func withEnv(f func()) {
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_DEFAULT_REGION")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()

	os.Setenv("AWS_ACCESS_KEY_ID", "FOO")
	os.Setenv("AWS_DEFAULT_REGION", "BAR")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "BAZ")

	f()
}

func withMocks(errorable bool, msg *sqs.Message, f func(*mockProducer)) func() {
	origClient := ssqs.DefaultClient
	origSendMsg := sendMessage

	return func() {
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
}
