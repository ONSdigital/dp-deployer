// Package ssqs provides a super simple AWS SQS consumer.
package ssqs

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
)

// DefaultClient returns a new SQS client.
var DefaultClient = func(config aws.Config) sqsiface.ClientAPI {
	return sqs.New(config)
}

// Consumer represents a consumer.
type Consumer struct {
	client   sqsiface.ClientAPI
	finish   chan struct{}
	Errors   chan error
	Messages chan *Message
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
	PollDuration      int64
	URL               string
	VisibilityTimeout int64
}

// New creates and returns a consumer.
func New(q *Queue) (*Consumer, error) {
	c, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, err
	}

	return &Consumer{
		client:   DefaultClient(c),
		finish:   make(chan struct{}, 1),
		Errors:   make(chan error, 1),
		Messages: make(chan *Message, 1),
		Queue:    q,
	}, nil
}

// Close closes a consumer.
func (c *Consumer) Close() {
	c.finish <- struct{}{}
}

// Delete deletes a message from the queue.
func (c *Consumer) Delete(m *Message) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      &c.Queue.URL,
		ReceiptHandle: &m.Receipt,
	}

	if _, err := c.client.DeleteMessageRequest(input).Send(context.Background()); err != nil {
		return err
	}
	return nil
}

// Start starts a consumer.
func (c *Consumer) Start() {
	input := &sqs.ReceiveMessageInput{
		QueueUrl:          &c.Queue.URL,
		VisibilityTimeout: &c.Queue.VisibilityTimeout,
		WaitTimeSeconds:   &c.Queue.PollDuration,
	}

	for {
		select {
		case <-c.finish:
			return
		default:
			c.receive(input)
		}
	}
}

func (c *Consumer) receive(input *sqs.ReceiveMessageInput) {
	r, err := c.client.ReceiveMessageRequest(input).Send(context.Background())
	if err != nil {
		c.Errors <- err
		return
	}
	for _, v := range r.Messages {
		c.Messages <- &Message{Body: *v.Body, ID: *v.MessageId, Receipt: *v.ReceiptHandle}
	}
}
