package ingestion

import (
	"context"
	"errors"
	"testing"

	"observability-hub/internal/secrets"
	"observability-hub/internal/worker/store"
)

type mockTask struct {
	name   string
	runErr error
}

func (m *mockTask) Name() string { return m.name }
func (m *mockTask) Run(ctx context.Context, s *store.Store, secretStore secrets.SecretStore) error {
	return m.runErr
}

func TestRunTask(t *testing.T) {
	tests := []struct {
		name    string
		task    *mockTask
		wantErr bool
	}{
		{
			name:    "Success",
			task:    &mockTask{name: "t1", runErr: nil},
			wantErr: false,
		},
		{
			name:    "Failure",
			task:    &mockTask{name: "t2", runErr: errors.New("err")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunTask(context.Background(), tt.task, &store.Store{}, &mockSecretStore{})
			if (err != nil) != tt.wantErr {
				t.Errorf("RunTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
