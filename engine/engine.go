// Package engine provides functionality for creating and running an engine.
package engine

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/LloydGriffiths/ssqs"
	"github.com/ONSdigital/go-ns/log"
	"github.com/cenkalti/backoff"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/sqs"
)

var wg sync.WaitGroup

// BackoffStrategy is the backoff strategy used when attempting retryable errors.
var BackoffStrategy = func() backoff.BackOff {
	return &backoff.ExponentialBackOff{
		Clock:               backoff.SystemClock,
		InitialInterval:     5 * time.Second,
		MaxInterval:         10 * time.Second,
		MaxElapsedTime:      300 * time.Second,
		Multiplier:          backoff.DefaultMultiplier,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
	}
}

// ErrHandler is the handler function applied to an error.
var ErrHandler = func(err error) { log.Error(err, nil) }

// Engine represents an engine.
type Engine struct {
	config   *Config
	consumer *ssqs.Consumer
	handlers map[string]HandlerFunc
	producer *sqs.SQS
}

// Message represents a message that has been consumed.
type Message struct {
	Artifact string
	Bucket   string
	Service  string
	Type     string
}

type response struct {
	Error   *string `json:"Error,omitempty"`
	ID      string
	Success bool
}

// HandlerFunc represents a function that is applied to a consumed message.
type HandlerFunc func(context.Context, *Message) error

// New returns a new engine.
func New(c *Config, hs map[string]HandlerFunc) (*Engine, error) {
	if len(c.ConsumerQueue) < 1 {
		return nil, ErrMissingConsumerQueue
	}
	if len(c.ConsumerQueueURL) < 1 {
		return nil, ErrMissingConsumerQueueURL
	}
	if len(c.ProducerQueue) < 1 {
		return nil, ErrMissingProdocerQueue
	}
	if len(c.Region) < 1 {
		return nil, ErrMissingRegion
	}

	a, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}

	return &Engine{
		config:   c,
		handlers: hs,
		producer: sqs.New(a, aws.Regions[c.Region]),
		consumer: ssqs.New(&ssqs.Queue{
			Name:              c.ConsumerQueue,
			Region:            c.Region,
			URL:               c.ConsumerQueueURL,
			VisibilityTimeout: 1800, // 30 minutes
		}),
	}, nil
}

// Start starts the queue consumer and applies a given handler function to each
// message that is consumed. Once the message has successfully been handled, we
// attempt to write the result of the handler function to an outbound queue. If
// the result is written successfully, the message that was originally consumed
// is removed from the queue.
func (e *Engine) Start(ctx context.Context) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.consumer.Start()
	}()
	e.run(ctx)
}

func (e *Engine) run(ctx context.Context) {
	for {
		select {
		case err := <-e.consumer.Errors:
			ErrHandler(err)
		case msg := <-e.consumer.Messages:
			wg.Add(1)
			go func(ctx context.Context, msg *ssqs.Message) {
				defer wg.Done()
				e.handle(ctx, msg)
			}(ctx, msg)
		case <-ctx.Done():
			log.Info("halting consumer", nil)
			e.consumer.Close()
			log.Info("waiting for handlers", nil)
			wg.Wait()
			return
		default:
		}
	}
}

func (e *Engine) handle(ctx context.Context, msg *ssqs.Message) {
	var err error

	backOff := backoff.WithContext(BackoffStrategy(), ctx)
	success := true

	var m Message
	if err = json.Unmarshal([]byte(msg.Body), &m); err != nil {
		success = false
		ErrHandler(err)
		goto PostHandle
	}
	if h, ok := e.handlers[m.Type]; !ok {
		success = false
		ErrHandler(&MissingHandlerError{m.Type})
	} else if err = h(ctx, &m); err != nil {
		success = false
		ErrHandler(err)
	}

PostHandle:
	result := &response{ID: msg.ID, Success: success}
	if err != nil {
		errs := err.Error()
		result.Error = &errs
	}
	backoff.RetryNotify(e.reply(result), backOff, func(err error, t time.Duration) { ErrHandler(err) })
	backoff.RetryNotify(e.delete(msg), backOff, func(err error, t time.Duration) { ErrHandler(err) })
}

func (e *Engine) delete(msg *ssqs.Message) func() error {
	return func() error { return e.consumer.Delete(msg) }
}

func (e *Engine) reply(res *response) func() error {
	return func() error {
		j, err := json.Marshal(res)
		if err != nil {
			return err
		}
		q, err := e.producer.GetQueue(e.config.ProducerQueue)
		if err != nil {
			return err
		}
		if _, err := q.SendMessage(string(j)); err != nil {
			return err
		}
		return nil
	}
}
