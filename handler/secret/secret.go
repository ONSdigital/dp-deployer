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

// AbortedError is an error implementation that includes the id of the aborted message.
type AbortedError struct {
	ID string
}

func (e *AbortedError) Error() string {
	return "aborted updating secrets for message"
}

var keyReader func() (io.Reader, error)

// HTTPClient is the default http client.
var HTTPClient = &http.Client{Timeout: time.Second * 10}

// Config represents the configuration for a secret.
type Config struct {
	// PrivateKeyPath is the path of the private key file.
	PrivateKeyPath string
	// Region is the region in which the secret artifacts bucket resides.
	Region string
}

// Secret represents a secret.
type Secret struct {
	entities        openpgp.EntityList
	s3Client        *s3.S3
	vault           *api.Logical
	vaultHTTPClient *http.Client
}

// New returns a new secret.
func New(c *Config) (*Secret, error) {
	if keyReader == nil {
		keyReader = fsKeyReader(c.PrivateKeyPath)
	}

	r, err := keyReader()
	if err != nil {
		return nil, err
	}
	e, err := entityList(r)
	if err != nil {
		return nil, err
	}
	a, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}

	vaultc := api.DefaultConfig()
	v, err := api.NewClient(vaultc)
	if err != nil {
		return nil, err
	}

	return &Secret{
		entities:        e,
		s3Client:        s3.New(a, aws.Regions[c.Region], HTTPClient),
		vault:           v.Logical(),
		vaultHTTPClient: vaultc.HttpClient,
	}, nil
}

// Handler handles secret messages that are delegated by the engine.
func (s *Secret) Handler(ctx context.Context, msg *engine.Message) error {
	for _, artifact := range msg.Artifacts {
		select {
		case <-ctx.Done():
			log.InfoC(msg.ID, "bailing on updating secrets", nil)
			return &AbortedError{ID: msg.ID}
		default:
			a, err := s.s3Client.Bucket(msg.Bucket).Get(artifact)
			if err != nil {
				return err
			}
			d, err := s.decryptMessage(a)
			if err != nil {
				return err
			}
			log.TraceC(msg.ID, "writing secret", log.Data{"artifact": artifact})
			if err := s.write(pathFor(artifact), d); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Secret) decryptMessage(message []byte) ([]byte, error) {
	a, err := dearmorMessage(bytes.NewReader(message))
	if err != nil {
		return nil, err
	}
	m, err := openpgp.ReadMessage(a.Body, s.entities, nil, nil)
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

func fsKeyReader(path string) func() (io.Reader, error) {
	return func() (io.Reader, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		return f, nil
	}
}

func entityList(reader io.Reader) (openpgp.EntityList, error) {
	if v, ok := reader.(io.Closer); ok {
		defer v.Close()
	}

	b, err := dearmorMessage(reader)
	if err != nil {
		return nil, err
	}
	e, err := openpgp.ReadEntity(packet.NewReader(b.Body))
	if err != nil {
		return nil, err
	}
	return openpgp.EntityList{e}, nil
}

func dearmorMessage(reader io.Reader) (*armor.Block, error) {
	b, err := armor.Decode(reader)
	if err != nil {
		return nil, err
	}
	return b, nil
}
