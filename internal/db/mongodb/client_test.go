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
	tests := []struct {
		name    string
		envVar  string
		mockErr error
		wantErr bool
	}{
		{
			name:    "URI Failure",
			envVar:  "",
			wantErr: true,
		},
		{
			name:    "Connect Failure",
			envVar:  "mongodb://localhost:27017",
			mockErr: errors.New("mongo connect failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				os.Setenv("MONGO_URI", tt.envVar)
			} else {
				os.Unsetenv("MONGO_URI")
			}
			defer os.Unsetenv("MONGO_URI")

			mock := &simpleMockStore{values: map[string]string{"uri": tt.envVar}}

			originalConnect := mongoConnect
			defer func() { mongoConnect = originalConnect }()
			mongoConnect = func(opts ...*options.ClientOptions) (*mongo.Client, error) {
				return nil, tt.mockErr
			}

			_, err := ConnectMongo(mock)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConnectMongo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetMongoURI(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		mockVals map[string]string
		want     string
		wantErr  bool
	}{
		{
			name:    "Missing Env",
			envVar:  "",
			wantErr: true,
		},
		{
			name:   "With Secret Store",
			envVar: "",
			mockVals: map[string]string{
				"uri": "mongodb://user:pass@localhost:27017",
			},
			want:    "mongodb://user:pass@localhost:27017",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				os.Setenv("MONGO_URI", tt.envVar)
			} else {
				os.Unsetenv("MONGO_URI")
			}
			defer os.Unsetenv("MONGO_URI")

			mock := &simpleMockStore{values: tt.mockVals}
			uri, err := GetMongoURI(mock)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMongoURI() error = %v, wantErr %v", err, tt.wantErr)
			}
			if uri != tt.want {
				t.Errorf("GetMongoURI() got = %v, want %v", uri, tt.want)
			}
		})
	}
}

func TestNewMongoStore(t *testing.T) {
	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{
			name:    "Connection Failure",
			mockErr: errors.New("mongo connect failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &simpleMockStore{values: map[string]string{"uri": "mongodb://localhost:27017"}}

			originalConnect := mongoConnect
			defer func() { mongoConnect = originalConnect }()
			mongoConnect = func(opts ...*options.ClientOptions) (*mongo.Client, error) {
				return nil, tt.mockErr
			}

			_, err := NewMongoStore(mock)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMongoStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMongoStore_Helpers(t *testing.T) {
	store := &MongoStore{
		Client: nil,
	}

	tests := []struct {
		name    string
		testFn  func(t *testing.T)
		wantErr bool
	}{
		{
			name: "Close Nil Client",
			testFn: func(t *testing.T) {
				err := store.Close(context.Background())
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "Find Nil Client",
			testFn: func(t *testing.T) {
				err := store.Find(context.Background(), "test-op", "observability-hub/internal/db", "coll", nil, nil, 10)
				if err == nil {
					t.Error("Expected error for nil client, got nil")
				}
			},
			wantErr: true,
		},
		{
			name: "UpdateByID Nil Client",
			testFn: func(t *testing.T) {
				err := store.UpdateByID(context.Background(), "test-op", "observability-hub/internal/db", "coll", "507f1f77bcf86cd799439011", nil)
				if err == nil {
					t.Error("Expected error for nil client, got nil")
				}
			},
			wantErr: true,
		},
		{
			name: "UpdateByID Invalid Hex",
			testFn: func(t *testing.T) {
				err := store.UpdateByID(context.Background(), "test-op", "test-db", "test-coll", "invalid-hex", nil)
				if err == nil {
					t.Error("Expected error for invalid hex ID, got nil")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFn(t)
		})
	}
}

func TestMockMongoStore(t *testing.T) {
	tests := []struct {
		name    string
		mock    *MockMongoStore
		wantErr bool
		errMsg  string
	}{
		{
			name: "Populated Mock",
			mock: &MockMongoStore{
				FindFn: func(ctx context.Context, opName, collection string, filter any, results any, limit int64) error {
					return errors.New("mock find")
				},
				UpdateByIDFn: func(ctx context.Context, opName, collection string, id string, update any) error {
					return errors.New("mock update")
				},
				CloseFn: func(ctx context.Context) error {
					return errors.New("mock close")
				},
			},
			wantErr: true,
			errMsg:  "mock",
		},
		{
			name:    "Nil Mock",
			mock:    &MockMongoStore{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name == "Populated Mock" {
				if err := tt.mock.Find(ctx, "op", "coll", nil, nil, 0); err == nil || err.Error() != "mock find" {
					t.Errorf("Expected mock find error, got %v", err)
				}
				if err := tt.mock.UpdateByID(ctx, "op", "coll", "id", nil); err == nil || err.Error() != "mock update" {
					t.Errorf("Expected mock update error, got %v", err)
				}
				if err := tt.mock.Close(ctx); err == nil || err.Error() != "mock close" {
					t.Errorf("Expected mock close error, got %v", err)
				}
			} else {
				if err := tt.mock.Find(ctx, "op", "coll", nil, nil, 0); err != nil {
					t.Errorf("Expected nil error for default mock find, got %v", err)
				}
				if err := tt.mock.UpdateByID(ctx, "op", "coll", "id", nil); err != nil {
					t.Errorf("Expected nil error for default mock update, got %v", err)
				}
				if err := tt.mock.Close(ctx); err != nil {
					t.Errorf("Expected nil error for default mock close, got %v", err)
				}
			}
		})
	}
}
