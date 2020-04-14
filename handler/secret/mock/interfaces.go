package secret

//go:generate moq -out ./mock/vault.go -pkg mock . VaultClient

// VaultClient is an interface to represent methods called to action upon Vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
}
