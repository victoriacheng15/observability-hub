package mongodb

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestMongoStore_OTelAttributes(t *testing.T) {
	// Setup test exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)
	otel.SetTracerProvider(tp)

	// Mock client
	client, _ := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:1"))
	store := &MongoStore{
		Client: client,
		user:   "test-mongo-user",
	}
	ctx := context.Background()

	t.Run("Verify Find Attributes", func(t *testing.T) {
		exporter.Reset()
		filter := map[string]string{"key": "value"}

		// This will fail because localhost:1 is not mongo, but the attributes should still be set before the fail
		_ = store.Find(ctx, "test-find", "db", "coll", filter, nil, 5)

		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("Expected span, got none")
		}

		attrs := make(map[string]string)
		for _, a := range spans[0].Attributes {
			attrs[string(a.Key)] = a.Value.AsString()
		}

		expected := map[string]string{
			"db.system":     "mongodb",
			"db.name":       "db",
			"db.collection": "coll",
			"db.user":       "test-mongo-user",
			"db.statement":  `{"key":"value"}`,
		}

		for k, v := range expected {
			if attrs[k] != v {
				t.Errorf("Expected attribute %s=%s, got %s", k, v, attrs[k])
			}
		}
	})
}

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
		{
			name:    "Invalid URI",
			envVar:  "mongodb:// ", // Space is invalid in URI
			wantErr: true,
		},
		{
			name:    "Ping Failure",
			envVar:  "mongodb://localhost:1", // Invalid port/no service
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
			if tt.mockErr != nil {
				mongoConnect = func(opts ...*options.ClientOptions) (*mongo.Client, error) {
					return nil, tt.mockErr
				}
			} else if tt.name == "Ping Failure" {
				// Use short timeout for faster test
				mongoConnect = func(opts ...*options.ClientOptions) (*mongo.Client, error) {
					// Apply short timeout via options if possible, but easier to just use real Connect
					// which will fail Ping because port 1 is unlikely to have mongo.
					return originalConnect(options.Client().ApplyURI(tt.envVar).SetConnectTimeout(10 * time.Millisecond))
				}
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
		{
			name:    "With Env Fallback",
			envVar:  "mongodb://env:pass@localhost:27017",
			want:    "mongodb://env:pass@localhost:27017",
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
		{
			name: "Find Broken Client",
			testFn: func(t *testing.T) {
				// Use a real client that is NOT connected to any mongo.
				// mongo.Connect with localhost:1 will probably not connect but will return a client.
				client, _ := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:1"))
				storeWithClient := &MongoStore{Client: client}
				err := storeWithClient.Find(context.Background(), "test-op", "db", "coll", nil, nil, 10)
				if err == nil {
					t.Error("Expected error for broken client, got nil")
				}
			},
			wantErr: true,
		},
		{
			name: "UpdateByID Broken Client",
			testFn: func(t *testing.T) {
				client, _ := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:1"))
				storeWithClient := &MongoStore{Client: client}
				err := storeWithClient.UpdateByID(context.Background(), "test-op", "db", "coll", "507f1f77bcf86cd799439011", nil)
				if err == nil {
					t.Error("Expected error for broken client, got nil")
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
