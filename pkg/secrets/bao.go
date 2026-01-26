package secrets

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/vault/api"
)

// BaoProvider implements the SecretStore interface for OpenBao.
type BaoProvider struct {
	client *api.Client
}

// NewBaoProvider initializes a new OpenBao client using environment variables.
// It expects BAO_ADDR and BAO_TOKEN to be set.
func NewBaoProvider() (*BaoProvider, error) {
	config := api.DefaultConfig()

	// Use BAO_ADDR if set, otherwise fallback to VAULT_ADDR logic in SDK
	if addr := os.Getenv("BAO_ADDR"); addr != "" {
		config.Address = addr
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create openbao client: %w", err)
	}

	// Use BAO_TOKEN if set
	if token := os.Getenv("BAO_TOKEN"); token != "" {
		client.SetToken(token)
	}

	return &BaoProvider{client: client}, nil
}

// GetSecret retrieves a secret from OpenBao at the given path and key.
// It follows the KV V2 secret engine format (secret/data/...).
// If the secret is missing or the client fails, it returns the fallback value.
func (b *BaoProvider) GetSecret(path, key, fallback string) string {
	if b.client == nil {
		return fallback
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secret, err := b.client.KVv2("secret").Get(ctx, path)
	if err != nil {
		return fallback
	}

	if val, ok := secret.Data[key].(string); ok {
		return val
	}

	return fallback
}

// Close is a placeholder for cleaning up resources if needed.
func (b *BaoProvider) Close() error {
	return nil
}
