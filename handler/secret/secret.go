package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"strings"

	// "golang.org/x/crypto/openpgp"
	// "golang.org/x/crypto/openpgp/armor"
	// "golang.org/x/crypto/openpgp/packet"

	"github.com/ONSdigital/dp-deployer/crypto/openpgp"
	"github.com/ONSdigital/dp-deployer/crypto/openpgp/armor"
	"github.com/ONSdigital/dp-deployer/crypto/openpgp/packet"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/s3"
	"github.com/ONSdigital/log.go/v2/log"
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
			log.Error(ctx, "bailing on updating secrets", errors.New("bailing on updating secrets"))
			return &AbortedError{ID: msg.ID}
		default:
			log.Info(ctx, "handling artifact", log.Data{"artifact": artifact})
			b, _, err := s.s3Client.Get(artifact)
			if err != nil {
				log.Error(ctx, "Secret-Handler, s.s3Client.Get(artifact) error", err)
				return err
			}
			// Make sure to close the body when done with it for S3 GetObject APIs or
			// will leak connections.
			defer b.Close()

			d, err := s.decryptMessage(b)
			if err != nil {
				log.Error(ctx, "Secret-Handler, s.decryptMessage(b) error", err)
				return err
			}
			log.Info(ctx, "writing secret", log.Data{"artifact": artifact})
			if err := s.write(pathFor(artifact), d); err != nil {
				log.Error(ctx, "Secret-Handler, s.write(pathFor) error", err)
				return err
			}
		}
	}
	return nil
}

func (s *Secret) decryptMessage(message io.Reader) ([]byte, error) {
	a, err := dearmorMessage(message)
	if err != nil {
		log.Error(context.Background(), "Secret-decryptMessage, dearmorMessage() error", err)
		return nil, err
	}
	m, err := openpgp.ReadMessage(a.Body, s.entities, nil, nil)
	if err != nil {
		log.Error(context.Background(), "Secret-decryptMessage, openpgp.ReadMessage() error", err)
		return nil, err
	}
	d, err := ioutil.ReadAll(m.UnverifiedBody)
	if err != nil {
		log.Error(context.Background(), "Secret-decryptMessage, ioutil.ReadAll() error", err)
		return nil, err
	}
	return d, nil
}

func (s *Secret) write(path string, secret []byte) error {
	var j map[string]interface{}
	if err := json.Unmarshal(secret, &j); err != nil {
		log.Error(context.Background(), "Secret-write, json.Unmarshal() error", err)
		return err
	}
	if err := s.vault.Write(fmt.Sprintf("secret/%s", path), j); err != nil {
		log.Error(context.Background(), "Secret-write, s.vault.Write() error", err)
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
		log.Error(context.Background(), "Secret-dearmorMessage, armor.Decode() error", err)
		return nil, err
	}
	return b, nil
}
