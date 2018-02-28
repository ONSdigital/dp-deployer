// Package engine provides functionality for creating and running an engine.
package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"

	"github.com/LloydGriffiths/ssqs"
	"github.com/ONSdigital/go-ns/log"
	"github.com/cenkalti/backoff"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/sqs"
)

var sendMessage func(string) error

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
var ErrHandler = func(messageID string, err error) { log.ErrorC(messageID, err, nil) }

// Config represents the configuration for an engine.
type Config struct {
	// ConsumerQueue is the name of the queue to consume messages from.
	ConsumerQueue string
	// ConsumerQueueURL is the URL of the queue to consume messages from.
	ConsumerQueueURL string
	// ProducerQueue is the name of the queue to produce messages to.
	ProducerQueue string
	// Region is the region of the queues.
	Region string
	// VerificationKey is used to verify the authenticity of a message.
	VerificationKey string
}

// Engine represents an engine.
type Engine struct {
	config    *Config
	consumer  *ssqs.Consumer
	keyring   openpgp.EntityList
	handlers  map[string]HandlerFunc
	producer  *sqs.SQS
	semaphore chan struct{}
	wg        sync.WaitGroup
}

// Message represents a message that has been consumed.
type Message struct {
	Artifacts []string
	Bucket    string
	ID        string `json:"-"`
	Service   string
	Type      string
}

// HandlerFunc represents a function that is applied to a consumed message.
type HandlerFunc func(context.Context, *Message) error

type response struct {
	Error   *responseError `json:"Error,omitempty"`
	ID      string
	Success bool
}

type responseError struct {
	Data    error
	Message string
}

// New returns a new engine.
func New(c *Config, hs map[string]HandlerFunc) (*Engine, error) {
	if len(c.ConsumerQueue) < 1 {
		return nil, ErrMissingConsumerQueue
	}
	if len(c.ConsumerQueueURL) < 1 {
		return nil, ErrMissingConsumerQueueURL
	}
	if len(c.ProducerQueue) < 1 {
		return nil, ErrMissingProducerQueue
	}
	if len(c.Region) < 1 {
		return nil, ErrMissingRegion
	}

	k, err := openpgp.ReadArmoredKeyRing(strings.NewReader(c.VerificationKey))
	if err != nil {
		return nil, err
	}

	a, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}

	e := &Engine{
		config:    c,
		keyring:   k,
		handlers:  hs,
		semaphore: make(chan struct{}, 50),
		producer:  sqs.New(a, aws.Regions[c.Region]),
		consumer: ssqs.New(&ssqs.Queue{
			Name:              c.ConsumerQueue,
			Region:            c.Region,
			URL:               c.ConsumerQueueURL,
			VisibilityTimeout: int64((time.Minute * 30).Seconds()),
		}),
	}

	if sendMessage == nil {
		sendMessage = e.sendMessage
	}

	return e, nil
}

// Start starts the queue consumer and applies a given handler function to each
// message that is consumed. Once the message has successfully been handled, we
// attempt to write the result of the handler function to an outbound queue. If
// the result is written successfully, the message that was originally consumed
// is removed from the queue.
func (e *Engine) Start(ctx context.Context) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.consumer.Start()
	}()
	e.run(ctx)
}

func (e *Engine) run(ctx context.Context) {
	for {
		select {
		case err := <-e.consumer.Errors:
			ErrHandler("", err)
		case msg := <-e.consumer.Messages:
			e.handle(ctx, msg)
		case <-ctx.Done():
			log.Info("halting consumer", nil)
			e.consumer.Close()
			log.Info("waiting for handlers", nil)
			e.wg.Wait()
			return
		default:
		}
	}
}

func (e *Engine) handle(ctx context.Context, rawMsg *ssqs.Message) {
	e.semaphore <- struct{}{}
	e.wg.Add(1)

	go func() {
		defer func() {
			e.wg.Done()
			<-e.semaphore
		}()

		m, err := e.verifyMessage(rawMsg)
		if err != nil {
			e.postHandle(ctx, rawMsg, err)
			return
		}

		engMsg := Message{ID: rawMsg.ID}
		if err := json.Unmarshal(m, &engMsg); err != nil {
			e.postHandle(ctx, rawMsg, err)
			return
		}
		if _, ok := e.handlers[engMsg.Type]; !ok {
			e.postHandle(ctx, rawMsg, &MissingHandlerError{engMsg.Type})
			return
		}
		if err := e.handlers[engMsg.Type](ctx, &engMsg); err != nil {
			e.postHandle(ctx, rawMsg, err)
			return
		}

		e.postHandle(ctx, rawMsg, nil)
	}()
}

func (e *Engine) postHandle(ctx context.Context, msg *ssqs.Message, err error) {
	if err != nil {
		ErrHandler(msg.ID, err)
	}

	result := &response{ID: msg.ID, Success: err == nil}
	if err != nil {
		result.Error = &responseError{Data: err, Message: err.Error()}
	}

	backoff.RetryNotify(
		e.reply(result),
		backoff.WithContext(BackoffStrategy(), ctx),
		func(err error, t time.Duration) { ErrHandler(msg.ID, err) },
	)
	backoff.RetryNotify(
		e.delete(msg),
		backoff.WithContext(BackoffStrategy(), ctx),
		func(err error, t time.Duration) { ErrHandler(msg.ID, err) },
	)
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
		if err := sendMessage(string(j)); err != nil {
			return err
		}
		return nil
	}
}

func (e *Engine) sendMessage(body string) error {
	q, err := e.producer.GetQueue(e.config.ProducerQueue)
	if err != nil {
		return err
	}
	if _, err := q.SendMessage(body); err != nil {
		return err
	}
	return nil
}

func (e *Engine) verifyMessage(rawMsg *ssqs.Message) ([]byte, error) {
	decoded, _ := clearsign.Decode([]byte(rawMsg.Body))
	if decoded == nil {
		return nil, &InvalidBlockError{rawMsg.ID}
	}
	if _, err := openpgp.CheckDetachedSignature(e.keyring, bytes.NewReader(decoded.Bytes), decoded.ArmoredSignature.Body); err != nil {
		return nil, err
	}
	return decoded.Plaintext, nil
}
