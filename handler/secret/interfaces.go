package secret

import vaultapi "github.com/hashicorp/vault/api"

//go:generate moq -out ./mock/vault.go -pkg mock . VaultClient

// VaultClient is an interface to represent methods called to action upon Vault
type VaultClient interface {
	Write(path string, data map[string]interface{}) (*vaultapi.Secret, error)
}
