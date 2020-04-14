package secret

//go:generate moq -out vaultmock_test.go . VaultClient

// VaultClient is an interface to represent methods called to action upon Vault
type VaultClient interface {
	Write(path string, data map[string]interface{}) error
}
