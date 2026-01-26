package secrets

// SecretStore defines the interface for retrieving sensitive configuration.
type SecretStore interface {
	// GetSecret retrieves a secret by path and key.
	// If the secret store is unavailable, it returns the fallback value.
	GetSecret(path, key, fallback string) string

	// Close cleans up any active connections to the secret store.
	Close() error
}
