package secret

import (
	"context"
	"net/http"
	"time"

	"github.com/ONSdigital/dp-ci/awdry/engine"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

// Secret represents a secret.
type Secret struct {
	client *s3.S3
}

// New returns a new secret.
func New(c *Config) (*Secret, error) {
	a, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}

	return &Secret{client: s3.New(a, aws.Regions[c.Region], http.DefaultClient)}, nil
}

// Handler handles secret messages that are delegated by the engine.
func (s *Secret) Handler(ctx context.Context, msg *engine.Message) error {
	for _, v := range msg.Artifacts {
		b, err := s.client.Bucket(msg.Bucket).Get(v)
		if err != nil {
			return err
		}
		// FIXME need to decrypt the secret
		if err := s.Write(b); err != nil {
			return err
		}
	}

	return nil
}

func (s *Secret) Write(bytes []byte) error {
	return nil
}
