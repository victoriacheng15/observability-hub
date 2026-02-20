package main

import (
	"context"
	"errors"
	"testing"

	"db/mongodb"
	"db/postgres"
	"secrets"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockSecretStore struct{}

func (m *mockSecretStore) GetSecret(path, key, fallback string) string { return fallback }
func (m *mockSecretStore) Close() error                                { return nil }

func TestApp_Run(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	tests := []struct {
		name      string
		secretErr error
		pgErr     error
		mongoErr  error
		wantErr   bool
	}{
		{"Success", nil, nil, nil, false},
		{"Secret Failure", errors.New("secret error"), nil, nil, true},
		{"Postgres Failure", nil, errors.New("pg error"), nil, true},
		{"Mongo Failure", nil, nil, errors.New("mongo error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Success" {
				mdb.ExpectTableCreation("reading_analytics")
				mdb.ExpectTableCreation("reading_sync_history")
				mdb.Mock.ExpectExec("INSERT INTO reading_sync_history").WillReturnResult(sqlmock.NewResult(1, 1))
			}

			app := &App{
				SecretProviderFn: func() (secrets.SecretStore, error) {
					return &mockSecretStore{}, tt.secretErr
				},
				PostgresConnFn: func(driver string, store secrets.SecretStore) (*postgres.ReadingStore, error) {
					return postgres.NewReadingStore(mdb.DB), tt.pgErr
				},
				MongoConnFn: func(store secrets.SecretStore) (MongoStoreAPI, error) {
					return &mongodb.MockMongoStore{}, tt.mongoErr
				},
			}

			err := app.Run(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApp_Sync(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()
	pgStore := postgres.NewReadingStore(mdb.DB)

	t.Run("Sync Success", func(t *testing.T) {
		docID := "507f1f77bcf86cd799439011"
		mStore := &mongodb.MockMongoStore{
			FetchFn: func(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error) {
				return []mongodb.ReadingDocument{
					{
						ID:        docID,
						Source:    "test",
						Type:      "cpu",
						Timestamp: "2026-01-01T00:00:00Z",
						Payload:   map[string]interface{}{"val": 1},
						Meta:      map[string]interface{}{"host": "localhost"},
					},
				}, nil
			},
		}

		mdb.ExpectTableCreation("reading_analytics")
		mdb.ExpectTableCreation("reading_sync_history")
		mdb.Mock.ExpectExec("INSERT INTO reading_analytics").WillReturnResult(sqlmock.NewResult(1, 1))
		mdb.Mock.ExpectExec("INSERT INTO reading_sync_history").WillReturnResult(sqlmock.NewResult(1, 1))

		app := &App{}
		err := app.Sync(context.Background(), pgStore, mStore)
		if err != nil {
			t.Errorf("Sync() failed: %v", err)
		}

		if err := mdb.Mock.ExpectationsWereMet(); err != nil {
			t.Errorf("mock expectations not met: %v", err)
		}
	})

	t.Run("Sync Fetch Error", func(t *testing.T) {
		mStore := &mongodb.MockMongoStore{
			FetchFn: func(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error) {
				return nil, errors.New("fetch error")
			},
		}

		mdb.ExpectTableCreation("reading_analytics")
		mdb.ExpectTableCreation("reading_sync_history")
		mdb.Mock.ExpectExec("INSERT INTO reading_sync_history").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "failure", 0, "fetch_from_mongo_failed: fetch error").
			WillReturnResult(sqlmock.NewResult(1, 1))

		app := &App{}
		err := app.Sync(context.Background(), pgStore, mStore)
		if err == nil {
			t.Error("Sync() expected error, got nil")
		}
	})
}
