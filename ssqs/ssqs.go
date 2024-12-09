// Package ssqs provides a super simple AWS SQS consumer.
package ssqs

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// DefaultClient returns a new SQS client.
var DefaultClient = func(q *Queue) *sqs.Client {
	return sqs.NewFromConfig(aws.NewConfig().Copy()) //session.New(), &aws.Config{Region: &q.Region})
}

// Consumer represents a consumer.
type Consumer struct {
	client   *sqs.Client
	finish   chan struct{}
	Errors   chan error
	Messages chan Message
	Queue    *Queue
}

// Message represents a queue message.
type Message struct {
	Body    string
	ID      string
	Receipt string
}

// Queue represents a consumers queue.
type Queue struct {
	Name              string
	PollDuration      int32
	Region            string
	URL               string
	VisibilityTimeout int32
}

// New creates and returns a consumer.
func New(q *Queue) *Consumer {
	return &Consumer{
		client:   DefaultClient(q),
		finish:   make(chan struct{}, 1),
		Errors:   make(chan error, 1),
		Messages: make(chan Message, 1),
		Queue:    q,
	}
}

// Close closes a consumer.
func (c *Consumer) Close() {
	c.finish <- struct{}{}
}

// Delete deletes a message from the queue.
func (c *Consumer) Delete(ctx context.Context, m *Message) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      &c.Queue.URL,
		ReceiptHandle: &m.Receipt,
	}

	if _, err := c.client.DeleteMessage(ctx, input); err != nil {
		return err
	}
	return nil
}

// Start starts a consumer.
func (c *Consumer) Start(ctx context.Context) {
	input := &sqs.ReceiveMessageInput{
		// AttributeNames:    []*string{&c.Queue.Name},
		QueueUrl:          &c.Queue.URL,
		VisibilityTimeout: c.Queue.VisibilityTimeout,
		WaitTimeSeconds:   c.Queue.PollDuration,
	}

	for {
		select {
		case <-c.finish:
			return
		default:
			c.receive(ctx, input)
		}
	}
}

func (c *Consumer) receive(ctx context.Context, input *sqs.ReceiveMessageInput) {
	r, err := c.client.ReceiveMessage(ctx, input)
	if err != nil {
		c.Errors <- err
		return
	}

	var count int

	// This processes up to 10 messages at a time from the amazon library
	for _, v := range r.Messages {
		c.Messages <- Message{Body: *v.Body, ID: *v.MessageId, Receipt: *v.ReceiptHandle}
		count++
	}
	if count == 0 {
		// without a delay this races around many 100's of times per second

		// The deployer is not a high performance application as some of the tasks
		// it is asked to do can take many seconds.
		// The deployer is not processing 10,000 of messages per second, sometimes it
		// may only be handling 10 messages per second.
		// If there are always messages in the queue, then this delay won't happen.
		time.Sleep(time.Millisecond * 500)
	}
}
