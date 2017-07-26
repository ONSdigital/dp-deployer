package secret

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		a, err := s.dearmorArtifact(msg.Bucket, artifact)
		if err != nil {
			return err
		}
		m, err := s.decryptArtifact(a.Body)
		if err != nil {
			return err
		}
		if err := s.write(pathFor(artifact), m); err != nil {
			return err
		}
	}

	return nil
}

func (s *Secret) dearmorArtifact(bucket, artifact string) (*armor.Block, error) {
	a, err := s.s3Client.Bucket(bucket).Get(artifact)
	if err != nil {
		return nil, err
	}
	b, err := armor.Decode(bytes.NewReader(a))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Secret) decryptArtifact(body io.Reader) ([]byte, error) {
	m, err := openpgp.ReadMessage(body, s.pgpEntities, nil, nil)
	if err != nil {
		return nil, err
	}
	d, err := ioutil.ReadAll(m.UnverifiedBody)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Secret) write(path string, secret []byte) error {
	log.Trace("writing secret", log.Data{"secret": path})

	var j map[string]interface{}
	if err := json.Unmarshal(secret, &j); err != nil {
		return err
	}
	if _, err := s.vault.Write(fmt.Sprintf("secret/%s", path), j); err != nil {
		return err
	}
	return nil
}

func pathFor(artifact string) string {
	return strings.Split(strings.Split(artifact, "/")[1], ".")[0]
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
