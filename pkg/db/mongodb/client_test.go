package mongodb

import (
	"context"
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

func TestMongoStore_Helpers(t *testing.T) {
	store := &MongoStore{
		Client: nil,
	}

	t.Run("Close Nil Client", func(t *testing.T) {
		err := store.Close(context.Background())
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Find Nil Client", func(t *testing.T) {
		err := store.Find(context.Background(), "test-op", "db", "coll", nil, nil, 10)
		if err == nil {
			t.Error("Expected error for nil client, got nil")
		}
	})

	t.Run("UpdateByID Nil Client", func(t *testing.T) {
		err := store.UpdateByID(context.Background(), "test-op", "db", "coll", "507f1f77bcf86cd799439011", nil)
		if err == nil {
			t.Error("Expected error for nil client, got nil")
		}
	})

	t.Run("UpdateByID Invalid Hex", func(t *testing.T) {
		err := store.UpdateByID(context.Background(), "test-op", "test-db", "test-coll", "invalid-hex", nil)
		if err == nil {
			t.Error("Expected error for invalid hex ID, got nil")
		}
	})
}

func TestMockMongoStore(t *testing.T) {
	t.Run("Populated Mock", func(t *testing.T) {
		mock := &MockMongoStore{
			FindFn: func(ctx context.Context, opName, collection string, filter any, results any, limit int64) error {
				return errors.New("mock find")
			},
			UpdateByIDFn: func(ctx context.Context, opName, collection string, id string, update any) error {
				return errors.New("mock update")
			},
			CloseFn: func(ctx context.Context) error {
				return errors.New("mock close")
			},
		}

		ctx := context.Background()
		if err := mock.Find(ctx, "op", "coll", nil, nil, 0); err == nil || err.Error() != "mock find" {
			t.Errorf("Expected mock find error, got %v", err)
		}
		if err := mock.UpdateByID(ctx, "op", "coll", "id", nil); err == nil || err.Error() != "mock update" {
			t.Errorf("Expected mock update error, got %v", err)
		}
		if err := mock.Close(ctx); err == nil || err.Error() != "mock close" {
			t.Errorf("Expected mock close error, got %v", err)
		}
	})

	t.Run("Nil Mock", func(t *testing.T) {
		mock := &MockMongoStore{}
		ctx := context.Background()
		if err := mock.Find(ctx, "op", "coll", nil, nil, 0); err != nil {
			t.Errorf("Expected nil error for default mock find, got %v", err)
		}
		if err := mock.UpdateByID(ctx, "op", "coll", "id", nil); err != nil {
			t.Errorf("Expected nil error for default mock update, got %v", err)
		}
		if err := mock.Close(ctx); err != nil {
			t.Errorf("Expected nil error for default mock close, got %v", err)
		}
	})
}
