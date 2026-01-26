package main

import (
	"os"
	"strings"
	"testing"

	"db"
)

// simpleMockStore implements secrets.SecretStore for testing
type simpleMockStore struct{}

func (m *simpleMockStore) GetSecret(path, key, fallback string) string {
	return fallback
}
func (m *simpleMockStore) Close() error { return nil }

func TestGetPostgresDSN_DatabaseURL(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	defer os.Unsetenv("DATABASE_URL")

	mock := &simpleMockStore{}

	expected := "postgres://user:pass@localhost:5432/db"
	result, err := db.GetPostgresDSN(mock)
	if err != nil {
		t.Errorf("GetPostgresDSN() error = %v", err)
	}
	if result != expected {
		t.Errorf("GetPostgresDSN() = %s, want %s", result, expected)
	}
}

func TestGetPostgresDSN_Parts(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("SERVER_DB_PASSWORD", "secret")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("SERVER_DB_PASSWORD")
	}()

	mock := &simpleMockStore{}

	result, err := db.GetPostgresDSN(mock)
	if err != nil {
		t.Errorf("GetPostgresDSN() error = %v", err)
	}
	if !strings.Contains(result, "host=localhost") ||
		!strings.Contains(result, "user=testuser") ||
		!strings.Contains(result, "dbname=testdb") ||
		!strings.Contains(result, "password=secret") {
		t.Errorf("GetPostgresDSN() returned unexpected string: %s", result)
	}
}

func TestGetPostgresDSN_MissingRequired(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("SERVER_DB_PASSWORD")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("SERVER_DB_PASSWORD")
	}()

	mock := &simpleMockStore{}

	_, err := db.GetPostgresDSN(mock)
	if err == nil {
		t.Error("GetPostgresDSN() expected error for missing env vars, got nil")
	}
}
