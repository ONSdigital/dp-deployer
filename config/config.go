package config

import (
	"encoding/json"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Configuration structure whiich holds information for configuring the deployer
type Configuration struct {
	ConsumerQueue              string        `envconfig:"CONSUMER_QUEUE"`
	ConsumerQueueURL           string        `envconfig:"CONSUMER_QUEUE_URL"`
	ProducerQueue              string        `envconfig:"PRODUCER_QUEUE"`
	VerificationKey            string        `envconfig:"VERIFICATION_KEY" json:"-"`
	DeploymentRoot             string        `envconfig:"DEPLOYMENT_ROOT"`
	NomadEndpoint              string        `envconfig:"NOMAD_ENDPOINT"`
	NomadToken                 string        `envconfig:"NOMAD_TOKEN" json:"-"`
	NomadCACert                string        `envconfig:"NOMAD_CA_CERT" json:"-"`
	NomadTLSSkipVerify         bool          `envconfig:"NOMAD_TLS_SKIP_VERIFY"`
	DeploymentTimeout          time.Duration `envconfig:"DEPLOYMENT_TIMEOUT"`
	BindAddr                   string        `envconfig:"BIND_ADDR"`
	HealthcheckInterval        time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthcheckCriticalTimeout time.Duration `envconfig:"HEALTHCHECK_CRTICAL_TIMEOUT"`
	PrivateKey                 string        `envconfig:"PRIVATE_KEY" json:"-"`
	VaultAddr                  string        `envconfig:"VAULT_ADDR"`
	VaultToken                 string        `envconfig:"VAULT_TOKEN"`
	AWSRegion                  string        `envconfig:"AWS_REGION"`
	SecretsBucketName          string        `envconfig:"SECRETS_BUCKET_NAME"`
	DeploymentsBucketName      string        `envconfig:"DEPLOYMENT_BUCKET_NAME"`
}

var cfg *Configuration

// Get the application and returns the configuration structure
func Get() (*Configuration, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Configuration{
		ConsumerQueue:              "",
		ConsumerQueueURL:           "",
		ProducerQueue:              "",
		VerificationKey:            "",
		DeploymentRoot:             "",
		NomadEndpoint:              "http://localhost:4646",
		NomadToken:                 "",
		NomadCACert:                "",
		NomadTLSSkipVerify:         false,
		DeploymentTimeout:          time.Second * 60 * 20,
		BindAddr:                   ":24300",
		HealthcheckInterval:        time.Second * 30,
		HealthcheckCriticalTimeout: time.Second * 10,
		PrivateKey:                 "",
		VaultAddr:                  "http://localhost:8200",
		VaultToken:                 "",
		AWSRegion:                  "eu-west-1",
		SecretsBucketName:          "",
		DeploymentsBucketName:      "",
	}
	return cfg, envconfig.Process("", cfg)
}

// String is implemented to prevent senstve fields being logged.
// The config is returned as JSON with sensitive fields omitted.
func (config Configuration) String() string {
	json, _ := json.Marshal(config)
	return string(json)
}
