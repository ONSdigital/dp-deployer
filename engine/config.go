package engine

// Config represents the queue configuration for an engine.
type Config struct {
	// ConsumerQueue is the name of the queue to consume messages from.
	ConsumerQueue string
	// ConsumerQueueURL is the URL of the queue to consume messages from.
	ConsumerQueueURL string
	// ProducerQueue is the name of the queue to produce messages to.
	ProducerQueue string
	// Region is the region of the queues.
	Region string
}
