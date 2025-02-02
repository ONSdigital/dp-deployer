// Package queue provides functionality for creating and running an engine.
// This queue package has not been implemented and the deployer still uses the old engine.
package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/message"
	"github.com/ONSdigital/dp-deployer/ssqs"
	"github.com/ONSdigital/dp-net/request"
	"github.com/ONSdigital/log.go/v2/log"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
)

// maxConcurrentHandlers limit on goroutines (each handling a message)
const maxConcurrentHandlers = 50

var sendMessage func(context.Context, string) error

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

// Queue represents a Queue.
type Queue struct {
	config    *config.Configuration
	consumer  *ssqs.Consumer
	keyring   openpgp.EntityList
	handlers  HandlerFunc
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
type HandlerFunc func(ctx context.Context, cfg config.Configuration, msg *message.MessageSQS) error

type response struct {
	Error   *responseError `json:"Error,omitempty"`
	ID      string
	Success bool
}

type responseError struct {
	Data    error
	Message string
}

// New returns a new queue.
func New(ctx context.Context, cfg *config.Configuration, hs HandlerFunc) (*Queue, error) {
	if len(cfg.ConsumerQueueNew) < 1 {
		return nil, ErrMissingConsumerQueue
	}
	if len(cfg.ConsumerQueueURLNew) < 1 {
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

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, err
	}

	if hs == nil {
		err = &MissingHandlerError{}
		return nil, err
	}

	q := &Queue{
		config:    cfg,
		keyring:   k,
		handlers:  hs,
		semaphore: make(chan struct{}, maxConcurrentHandlers),
		producer:  sqs.NewFromConfig(awsConfig),
		consumer: ssqs.New(&ssqs.Queue{
			Name:              cfg.ConsumerQueueNew,
			Region:            cfg.AWSRegion,
			URL:               cfg.ConsumerQueueURLNew,
			VisibilityTimeout: int64((time.Minute * 30).Seconds()),
		}),
	}

	if sendMessage == nil {
		sendMessage = q.sendMessage
	}

	return q, nil
}

// Start starts the queue consumer and applies a given handler function to each
// message that is consumed. Once the message has successfully been handled, we
// attempt to write the result of the handler function to an outbound queue. If
// the result is written successfully, the message that was originally consumed
// is removed from the queue.
func (q *Queue) Start(ctx context.Context) {
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		q.consumer.Start()
	}()
	q.run(ctx)
}

// Close ssqs queue
func (q *Queue) Close() {
	log.Info(context.Background(), "halting consumer")
	q.consumer.Close()
	log.Info(context.Background(), "waiting for handlers")
	q.wg.Wait()
}

func (q *Queue) run(ctx context.Context) {
	for {
		select {
		case err := <-q.consumer.Errors:
			ErrHandler(ctx, "received consumer error", err)
		case msg := <-q.consumer.Messages:
			reqCtx := request.WithRequestId(ctx, msg.ID)
			q.handle(reqCtx, &msg)
		case <-ctx.Done():
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (q *Queue) handle(ctx context.Context, rawMsg *ssqs.Message) {
	q.semaphore <- struct{}{}
	q.wg.Add(1)

	go func() {
		defer func() {
			q.wg.Done()
			<-q.semaphore
		}()

		m, err := q.verifyMessage(rawMsg)
		if err != nil {
			q.postHandle(ctx, rawMsg, err)
			return
		}

		queueMsg := message.MessageSQS{Job: rawMsg.ID} // replace this with messageSQS
		if err := json.Unmarshal(m, &queueMsg); err != nil {
			q.postHandle(ctx, rawMsg, err)
			return
		}

		if err := q.handlers(ctx, *q.config, &queueMsg); err != nil {
			q.postHandle(ctx, rawMsg, err)
			return
		}

		q.postHandle(ctx, rawMsg, nil)
	}()
}

func (q *Queue) postHandle(ctx context.Context, msg *ssqs.Message, err error) {
	if err != nil {
		ErrHandler(ctx, "post handle error", err)
	}

	result := &response{ID: msg.ID, Success: err == nil}
	if err != nil {
		result.Error = &responseError{Data: err, Message: err.Error()}
	}

	backoff.RetryNotify(
		q.reply(ctx, result),
		backoff.WithContext(BackoffStrategy(), ctx),
		func(err error, t time.Duration) { ErrHandler(ctx, "failed to send reply to sqs queue", err) },
	)
	backoff.RetryNotify(
		q.delete(msg),
		backoff.WithContext(BackoffStrategy(), ctx),
		func(err error, t time.Duration) { ErrHandler(ctx, "failed to delete message from sqs queue", err) },
	)
}

func (q *Queue) delete(msg *ssqs.Message) func() error {
	return func() error { return q.consumer.Delete(msg) }
}

func (q *Queue) reply(ctx context.Context, res *response) func() error {
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

func (q *Queue) sendMessage(ctx context.Context, body string) error {
	resultGet, err := q.producer.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &q.config.ProducerQueue,
	})
	if err != nil {
		return err
	}
	queueURL := resultGet.QueueUrl

	msgRes, err := q.producer.SendMessage(ctx, &sqs.SendMessageInput{
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

func (q *Queue) verifyMessage(rawMsg *ssqs.Message) ([]byte, error) {
	decoded, _ := clearsign.Decode([]byte(rawMsg.Body))
	if decoded == nil {
		return nil, &InvalidBlockError{rawMsg.ID}
	}
	if _, err := openpgp.CheckDetachedSignature(q.keyring, bytes.NewReader(decoded.Bytes), decoded.ArmoredSignature.Body); err != nil {
		return nil, err
	}
	return decoded.Plaintext, nil
}
