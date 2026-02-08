package postgres

import (
	"database/sql"
	"database/sql/driver"
	"errors"
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

// --- Mock SQL Driver ---

type mockDriver struct{}

func (d mockDriver) Open(name string) (driver.Conn, error) {
	if name == "fail-connect" {
		return nil, errors.New("connection failed")
	}
	return &mockConn{}, nil
}

type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (c *mockConn) Close() error                              { return nil }
func (c *mockConn) Begin() (driver.Tx, error)                 { return nil, nil }

func init() {
	sql.Register("mock-postgres", &mockDriver{})
}

func TestConnectPostgres(t *testing.T) {
	mock := &simpleMockStore{
		values: map[string]string{
			"host":               "localhost",
			"port":               "5432",
			"user":               "user",
			"dbname":             "db",
			"server_db_password": "pass",
		},
	}

	t.Run("Success", func(t *testing.T) {
		db, err := ConnectPostgres("mock-postgres", mock)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if db == nil {
			t.Fatal("Expected db instance, got nil")
		}
		defer db.Close()
	})

	t.Run("Unknown Driver", func(t *testing.T) {
		_, err := ConnectPostgres("unknown-driver", mock)
		if err == nil {
			t.Error("Expected error for unknown driver, got nil")
		}
	})

	t.Run("DSN Failure", func(t *testing.T) {
		emptyMock := &simpleMockStore{values: map[string]string{}}
		os.Unsetenv("SERVER_DB_PASSWORD")
		_, err := ConnectPostgres("mock-postgres", emptyMock)
		if err == nil {
			t.Error("Expected error due to missing credentials, got nil")
		}
	})
}

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

	expected := "host=bao-host port=30432 user=server password=env-password dbname=homelab sslmode=disable timezone=UTC"
	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}
