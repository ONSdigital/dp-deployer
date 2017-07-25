package secret

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"

	"github.com/ONSdigital/dp-ci/awdry/engine"
	"github.com/ONSdigital/go-ns/log"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/hashicorp/vault/api"
)

// Secret represents a secret.
type Secret struct {
	pgpEntities openpgp.EntityList
	s3Client    *s3.S3
	vault       *api.Logical
}

// New returns a new secret.
func New(c *Config) (*Secret, error) {
	e, err := entityList(c.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	a, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}
	v, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	return &Secret{
		pgpEntities: e,
		s3Client:    s3.New(a, aws.Regions[c.Region], http.DefaultClient),
		vault:       v.Logical(),
	}, nil
}

// Handler handles secret messages that are delegated by the engine.
func (s *Secret) Handler(ctx context.Context, msg *engine.Message) error {
	for _, artifact := range msg.Artifacts {
		a, err := s.s3Client.Bucket(msg.Bucket).Get(artifact)
		if err != nil {
			return err
		}
		b, err := armor.Decode(bytes.NewReader(a))
		if err != nil {
			return err
		}
		m, err := openpgp.ReadMessage(b.Body, s.pgpEntities, nil, nil)
		if err != nil {
			return err
		}
		d, err := ioutil.ReadAll(m.UnverifiedBody)
		if err != nil {
			return err
		}

		var j map[string]interface{}
		if err := json.Unmarshal(d, &j); err != nil {
			return err
		}

		ps := strings.Split(artifact, "/")
		fs := strings.Split(ps[1], ".")
		log.Trace("writing secret", log.Data{"object": artifact, "secret": fs[0]})

		if _, err := s.vault.Write(fmt.Sprintf("secret/%s", fs[0]), j); err != nil {
			return err
		}
	}

	return nil
}

func entityList(path string) (openpgp.EntityList, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := armor.Decode(f)
	if err != nil {
		return nil, err
	}
	e, err := openpgp.ReadEntity(packet.NewReader(b.Body))
	if err != nil {
		return nil, err
	}

	return openpgp.EntityList{e}, nil
}
