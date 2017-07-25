package secret

// Config represents the configuration for a secret.
type Config struct {
	// PrivateKeyPath is the path of the private key file.
	PrivateKeyPath string
	// Region is the region in which the secret artifacts bucket resides.
	Region string
}
