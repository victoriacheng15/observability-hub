package main

import (
	"context"
	"errors"
	"os"
	"testing"

	"db/mongodb"
	"db/postgres"
	"secrets"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockSecretStore struct{}

func (m *mockSecretStore) GetSecret(path, key, fallback string) string { return fallback }
func (m *mockSecretStore) Close() error                                { return nil }

func TestApp_Bootstrap(t *testing.T) {
	dbConn, _, _ := sqlmock.New()
	defer dbConn.Close()

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
			os.Setenv("APP_ENV", "test")
			defer os.Unsetenv("APP_ENV")

			app := &App{
				SecretProviderFn: func() (secrets.SecretStore, error) {
					return &mockSecretStore{}, tt.secretErr
				},
				PostgresConnFn: func(driver string, store secrets.SecretStore) (*postgres.ReadingStore, error) {
					return postgres.NewReadingStore(dbConn), tt.pgErr
				},
				MongoConnFn: func(store secrets.SecretStore) (*mongodb.MongoStore, error) {
					return &mongodb.MongoStore{}, tt.mongoErr
				},
			}

			err := app.Bootstrap(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
