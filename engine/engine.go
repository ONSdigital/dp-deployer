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

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/ssqs"
	"github.com/ONSdigital/dp-net/request"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// maxConcurrentHandlers limit on goroutines (each handling a message)
const maxConcurrentHandlers = 50

var sendMessage func(context.Context, string) error

var loadDefaultConfigFunc func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) = awsconfig.LoadDefaultConfig

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
var ErrHandler = func(ctx context.Context, event string, err error) {
	log.Error(ctx, event, err)
}

// Engine represents an engine.
type Engine struct {
	config    *config.Configuration
	consumer  *ssqs.Consumer
	keyring   openpgp.EntityList
	handlers  map[string]HandlerFunc
	producer  *sqs.Client
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
func New(ctx context.Context, cfg *config.Configuration, hs map[string]HandlerFunc) (*Engine, error) {
	if len(cfg.ConsumerQueue) < 1 {
		return nil, ErrMissingConsumerQueue
	}
	if len(cfg.ConsumerQueueURL) < 1 {
		return nil, ErrMissingConsumerQueueURL
	}
	if len(cfg.ProducerQueue) < 1 {
		return nil, ErrMissingProducerQueue
	}
	if len(cfg.AWSRegion) < 1 {
		return nil, ErrMissingRegion
	}

	k, err := openpgp.ReadArmoredKeyRing(strings.NewReader(cfg.VerificationKey))
	if err != nil {
		return nil, err
	}

	awsConfig, err := loadDefaultConfigFunc(ctx, awsconfig.WithRegion(cfg.AWSRegion))

	if err != nil {
		return nil, err
	}

	e := &Engine{
		config:    cfg,
		keyring:   k,
		handlers:  hs,
		semaphore: make(chan struct{}, maxConcurrentHandlers),
		producer:  sqs.NewFromConfig(awsConfig),
		consumer: ssqs.New(&ssqs.Queue{
			Name:              cfg.ConsumerQueue,
			Region:            cfg.AWSRegion,
			URL:               cfg.ConsumerQueueURL,
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

// Close ssqs queue
func (e *Engine) Close() {
	log.Info(context.Background(), "halting consumer")
	e.consumer.Close()
	log.Info(context.Background(), "waiting for handlers")
	e.wg.Wait()
}

func (e *Engine) run(ctx context.Context) {
	for {
		select {
		case err := <-e.consumer.Errors:
			ErrHandler(ctx, "received consumer error", err)
		case msg := <-e.consumer.Messages:
			reqCtx := request.WithRequestId(ctx, msg.ID)
			e.handle(reqCtx, &msg)
		case <-ctx.Done():
			return
		default:
			time.Sleep(100 * time.Millisecond)
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
			log.Error(ctx, "handle(), e.verifyMessage(rawMsg) error", err)
			e.postHandle(ctx, rawMsg, err)
			return
		}

		engMsg := Message{ID: rawMsg.ID}
		if err := json.Unmarshal(m, &engMsg); err != nil {
			log.Error(ctx, "handle(), json.Unmarshal() error", err)
			e.postHandle(ctx, rawMsg, err)
			return
		}

		var handlerFunc HandlerFunc
		var ok bool
		if handlerFunc, ok = e.handlers[engMsg.Type]; !ok {
			log.Error(ctx, "handle(), e.handlers[engMsg.Type] error", err)
			e.postHandle(ctx, rawMsg, &MissingHandlerError{engMsg.Type})
			return
		}
		if err := handlerFunc(ctx, &engMsg); err != nil {
			log.Error(ctx, "handle(), handlerFunc() error", err)
			e.postHandle(ctx, rawMsg, err)
			return
		}

		e.postHandle(ctx, rawMsg, nil)
	}()
}

func (e *Engine) postHandle(ctx context.Context, msg *ssqs.Message, err error) {
	if err != nil {
		ErrHandler(ctx, "post handle error", err)
	}

	result := &response{ID: msg.ID, Success: err == nil}
	if err != nil {
		result.Error = &responseError{Data: err, Message: err.Error()}
	}

	backoff.RetryNotify(
		e.reply(ctx, result),
		backoff.WithContext(BackoffStrategy(), ctx),
		func(err error, t time.Duration) { ErrHandler(ctx, "failed to send reply to sqs queue", err) },
	)
	backoff.RetryNotify(
		e.delete(msg),
		backoff.WithContext(BackoffStrategy(), ctx),
		func(err error, t time.Duration) { ErrHandler(ctx, "failed to delete message from sqs queue", err) },
	)
}

func (e *Engine) delete(msg *ssqs.Message) func() error {
	return func() error { return e.consumer.Delete(msg) }
}

func (e *Engine) reply(ctx context.Context, res *response) func() error {
	return func() error {
		j, err := json.Marshal(res)
		if err != nil {
			return err
		}
		if err := sendMessage(ctx, string(j)); err != nil {
			return err
		}
		return nil
	}
}

func (e *Engine) sendMessage(ctx context.Context, body string) error {

	resultGet, err := e.producer.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &e.config.ProducerQueue,
	})
	if err != nil {
		return err
	}
	queueURL := resultGet.QueueUrl

	msgRes, err := e.producer.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    queueURL,
		MessageBody: &body,
	})

	if err != nil {
		return err
	}
	if msgRes == nil {
		return errors.New("msgRes was nil")
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
