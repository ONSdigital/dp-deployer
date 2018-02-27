package engine

import (
	"errors"
)

var (
	// ErrMissingConsumerQueue is returned when the consumer queue name is missing.
	ErrMissingConsumerQueue = errors.New("missing consumer queue name")
	// ErrMissingConsumerQueueURL is returned when the consumer queue url is missing.
	ErrMissingConsumerQueueURL = errors.New("missing consumer queue url")
	// ErrMissingProducerQueue is returned when the producer queue name is missing.
	ErrMissingProducerQueue = errors.New("missing producer queue name")
	// ErrMissingRegion is returned when the queue region is missing.
	ErrMissingRegion = errors.New("missing queue region")
)

// ErrInvalidBlock is returned when a message isn't a clearsigned message.
type InvalidBlockError struct {
	MessageId string
}

func (e *InvalidBlockError) Error() string {
	return "invalid block in message"
}

// MissingHandlerError is an error implementation that includes a consumed message type.
type MissingHandlerError struct {
	MessageType string
}

func (e *MissingHandlerError) Error() string {
	return "missing handler for message"
}
