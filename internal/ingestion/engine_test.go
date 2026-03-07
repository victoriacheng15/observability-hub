package ingestion

import (
	"context"
	"errors"
	"testing"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/secrets"
)

type mockTask struct {
	name   string
	runErr error
}

func (m *mockTask) Name() string {
	return m.name
}

func (m *mockTask) Run(ctx context.Context, db *postgres.PostgresWrapper, secretStore secrets.SecretStore) error {
	return m.runErr
}

type mockSecretStore struct {
	secrets.SecretStore
}

func (m *mockSecretStore) GetSecret(path, key, fallback string) string {
	return fallback
}

func TestRunTask(t *testing.T) {
	tests := []struct {
		name    string
		task    *mockTask
		wantErr bool
	}{
		{
			name: "Success",
			task: &mockTask{
				name:   "test-task",
				runErr: nil,
			},
			wantErr: false,
		},
		{
			name: "Failure",
			task: &mockTask{
				name:   "test-task",
				runErr: errors.New("task failed"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			db := &postgres.PostgresWrapper{}
			secretStore := &mockSecretStore{}

			err := RunTask(ctx, tt.task, db, secretStore)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
