package s3

import "io"

//go:generate moq -out s3mock.go . Client

// Client is an interface to represent methods called to action upon S3
type Client interface {
	Get(key string) (io.ReadCloser, *int64, error)
}
