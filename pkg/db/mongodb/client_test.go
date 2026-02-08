package mongodb

import (
	"errors"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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
		mongoConnect = func(opts ...*options.ClientOptions) (*mongo.Client, error) {
			return nil, errors.New("mongo connect failed")
		}

		_, err := ConnectMongo(mock)
		if err == nil {
			t.Error("Expected error due to connection failure, got nil")
		}
	})
}

func TestGetMongoURI_MissingEnv(t *testing.T) {
	os.Unsetenv("MONGO_URI")
	mock := &simpleMockStore{}

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

func TestNewMongoStore(t *testing.T) {
	mock := &simpleMockStore{
		values: map[string]string{"uri": "mongodb://localhost:27017"},
	}

	originalConnect := mongoConnect
	defer func() { mongoConnect = originalConnect }()
	mongoConnect = func(opts ...*options.ClientOptions) (*mongo.Client, error) {
		return nil, errors.New("mongo connect failed")
	}

	_, err := NewMongoStore(mock)
	if err == nil {
		t.Error("Expected error due to connection failure in NewMongoStore, got nil")
	}
}
