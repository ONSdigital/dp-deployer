package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/s3"
	"github.com/ONSdigital/log.go/log"
)

// AbortedError is an error implementation that includes the id of the aborted message.
type AbortedError struct {
	ID string
}

func (e *AbortedError) Error() string {
	return "aborted updating secrets for message"
}

// Secret represents a secret.
type Secret struct {
	entities openpgp.EntityList
	s3Client s3.Client
	vault    VaultClient
}

// New returns a new secret.
func New(cfg *config.Configuration, vc VaultClient, secretsClient s3.Client) (*Secret, error) {
	e, err := entityList(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	return &Secret{
		entities: e,
		s3Client: secretsClient,
		vault:    vc,
	}, nil
}

// Handler handles secret messages that are delegated by the engine.
func (s *Secret) Handler(ctx context.Context, msg *engine.Message) error {
	for _, artifact := range msg.Artifacts {
		select {
		case <-ctx.Done():
			log.Event(ctx, "bailing on updating secrets", log.ERROR)
			return &AbortedError{ID: msg.ID}
		default:
			b, _, err := s.s3Client.Get(artifact)
			if err != nil {
				return err
			}
			d, err := s.decryptMessage(b)
			if err != nil {
				return err
			}
			log.Event(ctx, "writing secret", log.INFO, log.Data{"artifact": artifact})
			if err := s.write(pathFor(artifact), d); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Secret) decryptMessage(message io.Reader) ([]byte, error) {
	a, err := dearmorMessage(message)
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
	if err := s.vault.Write(fmt.Sprintf("secret/%s", path), j); err != nil {
		return err
	}
	return nil
}

func pathFor(artifact string) string {
	return strings.Split(strings.Split(artifact, "/")[1], ".")[0]
}

func entityList(privateKey string) (openpgp.EntityList, error) {
	b, err := dearmorMessage(strings.NewReader(privateKey))
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
