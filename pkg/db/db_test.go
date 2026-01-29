package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	// Simulate connection failure based on DSN content
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

// --- Tests ---

func TestConnectMongo(t *testing.T) {
	t.Run("URI Failure", func(t *testing.T) {
		mock := &simpleMockStore{values: map[string]string{}}
		os.Unsetenv("MONGO_URI")
		_, err := ConnectMongo(mock)
		if err == nil {
			t.Error("Expected error due to missing MONGO_URI, got nil")
		}
	})

	t.Run("Connect Failure", func(t *testing.T) {
		mock := &simpleMockStore{
			values: map[string]string{"uri": "mongodb://localhost:27017"},
		}

		// Mock mongoConnect to fail
		originalConnect := mongoConnect
		defer func() { mongoConnect = originalConnect }()
		mongoConnect = func(ctx context.Context, opts ...*options.ClientOptions) (*mongo.Client, error) {
			return nil, errors.New("mongo connect failed")
		}

		_, err := ConnectMongo(mock)
		if err == nil {
			t.Error("Expected error due to connection failure, got nil")
		}
	})
}

func TestConnectPostgres(t *testing.T) {
	mock := &simpleMockStore{
		values: map[string]string{
			"host":               "localhost",
			"port":               "5432",
			"user":               "user",
			"dbname":             "db",
			"server_db_password": "pass", // Ensure GetPostgresDSN succeeds
		},
	}

	t.Run("Success", func(t *testing.T) {
		// Mock driver "mock-postgres" ignores the DSN string content for connection,
		// but ConnectPostgres calls GetPostgresDSN first.
		// GetPostgresDSN returns a string like "host=...".
		// Our mockDriver.Open receives this string.

		// To distinguish fail case, we might need to manipulate what GetPostgresDSN returns?
		// But GetPostgresDSN logic is fixed.
		// Actually, sql.Open("mock-postgres", dsn).
		// mockDriver.Open(dsn) is called.

		db, err := ConnectPostgres("mock-postgres", mock)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if db == nil {
			t.Fatal("Expected db instance, got nil")
		}
		defer db.Close()
	})

	// To test failure, we need driver to fail.
	// But our driver uses the DSN string to decide.
	// GetPostgresDSN output is standard.
	// However, we can use a DIFFERENT driver name that is not registered?
	// sql.Open succeeds even if driver not registered? No, it panics or errors?
	// sql.Open returns error if driver not found.

	t.Run("Unknown Driver", func(t *testing.T) {
		_, err := ConnectPostgres("unknown-driver", mock)
		if err == nil {
			t.Error("Expected error for unknown driver, got nil")
		}
	})

	// DSN Failure
	t.Run("DSN Failure", func(t *testing.T) {
		emptyMock := &simpleMockStore{values: map[string]string{}}
		os.Unsetenv("SERVER_DB_PASSWORD") // Ensure fallback fails
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
	mock := &simpleMockStore{} // returns fallback

	uri, err := GetMongoURI(mock)
	if err == nil {
		t.Error("Expected error when MONGO_URI is missing, got nil")
	}
	if uri != "" {
		t.Errorf("Expected empty URI when MONGO_URI is missing, got %s", uri)
	}
}

func TestGetMongoURI_WithSecretStore(t *testing.T) {
	expected := "mongodb://user:pass@localhost:27017"
	mock := &simpleMockStore{
		values: map[string]string{
			"uri": expected,
		},
	}

	uri, err := GetMongoURI(mock)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if uri != expected {
		t.Errorf("Expected URI %q, got %q", expected, uri)
	}
}
