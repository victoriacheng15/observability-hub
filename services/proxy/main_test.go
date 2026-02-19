package main

import (
	"context"
	"errors"
	"os"
	"testing"

	"secrets"
)

type mockSecretStore struct{}

func (m *mockSecretStore) GetSecret(path, key, fallback string) string { return fallback }
func (m *mockSecretStore) Close() error                                { return nil }

func TestApp_Bootstrap(t *testing.T) {
	tests := []struct {
		name      string
		secretErr error
		wantErr   bool
	}{
		{"Success", nil, false},
		{"Secret Failure", errors.New("secret error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("APP_ENV", "test")
			defer os.Unsetenv("APP_ENV")

			app := &App{
				SecretProviderFn: func() (secrets.SecretStore, error) {
					return &mockSecretStore{}, tt.secretErr
				},
			}

			err := app.Bootstrap(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
