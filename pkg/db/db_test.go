package db

import (
	"os"
	"testing"
)

// simpleMockStore satisfies the secrets.SecretStore interface for testing
type simpleMockStore struct {
	values map[string]string
}

func (m *simpleMockStore) GetSecret(path, key, fallback string) string {
	if val, ok := m.values[key]; ok {
		return val
	}
	return fallback
}

func (m *simpleMockStore) Close() error { return nil }

func TestGetPostgresDSN_WithSecretStore(t *testing.T) {
	os.Unsetenv("DB_HOST")
	os.Setenv("SERVER_DB_PASSWORD", "env-password")

	mock := &simpleMockStore{
		values: map[string]string{
			"host": "bao-host",
		},
	}

	dsn, err := GetPostgresDSN(mock)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// host should come from mock ("bao-host")
	// password should come from env fallback ("env-password")
	// port should come from default fallback ("5432")
	expected := "host=bao-host port=5432 user=server password=env-password dbname=homelab sslmode=disable timezone=UTC"
	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}

func TestGetMongoURI_MissingEnv(t *testing.T) {
	os.Unsetenv("MONGO_URI")

	uri, err := GetMongoURI()
	if err == nil {
		t.Error("Expected error when MONGO_URI is missing, got nil")
	}
	if uri != "" {
		t.Errorf("Expected empty URI when MONGO_URI is missing, got %s", uri)
	}
}
