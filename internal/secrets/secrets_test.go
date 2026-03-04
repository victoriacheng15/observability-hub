package secrets

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/vault/api"
)

// mockKV implements the vaultKVv2 interface for unit testing.
type mockKV struct {
	data map[string]*api.KVSecret
	err  error
}

func (m *mockKV) Get(ctx context.Context, path string) (*api.KVSecret, error) {
	if m.err != nil {
		return nil, m.err
	}
	if secret, ok := m.data[path]; ok {
		return secret, nil
	}
	return nil, fmt.Errorf("secret not found")
}

func TestNewBaoProvider(t *testing.T) {
	// Save original env
	origAddr := os.Getenv("BAO_ADDR")
	origToken := os.Getenv("BAO_TOKEN")
	defer func() {
		os.Setenv("BAO_ADDR", origAddr)
		os.Setenv("BAO_TOKEN", origToken)
	}()

	tests := []struct {
		name  string
		addr  string
		token string
	}{
		{
			name:  "Custom Addr and Token",
			addr:  "http://localhost:8200",
			token: "test-token",
		},
		{
			name:  "Empty Env",
			addr:  "",
			token: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.addr != "" {
				os.Setenv("BAO_ADDR", tt.addr)
			} else {
				os.Unsetenv("BAO_ADDR")
			}
			if tt.token != "" {
				os.Setenv("BAO_TOKEN", tt.token)
			} else {
				os.Unsetenv("BAO_TOKEN")
			}

			provider, err := NewBaoProvider()
			if err != nil {
				t.Fatalf("NewBaoProvider() failed: %v", err)
			}
			if provider == nil {
				t.Fatal("Expected provider instance, got nil")
			}
		})
	}
}

func TestBaoProvider_GetSecret(t *testing.T) {
	tests := []struct {
		name     string
		mockData map[string]*api.KVSecret
		mockErr  error
		path     string
		key      string
		fallback string
		want     string
	}{
		{
			name: "Success",
			mockData: map[string]*api.KVSecret{
				"test/path": {
					Data: map[string]interface{}{
						"api_key": "secret-value",
					},
				},
			},
			path:     "test/path",
			key:      "api_key",
			fallback: "fallback",
			want:     "secret-value",
		},
		{
			name: "Missing Key",
			mockData: map[string]*api.KVSecret{
				"test/path": {
					Data: map[string]interface{}{
						"other_key": "value",
					},
				},
			},
			path:     "test/path",
			key:      "api_key",
			fallback: "fallback",
			want:     "fallback",
		},
		{
			name:     "Vault Error",
			mockErr:  errors.New("vault connection failed"),
			path:     "test/path",
			key:      "api_key",
			fallback: "fallback",
			want:     "fallback",
		},
		{
			name:     "Path Not Found",
			mockData: map[string]*api.KVSecret{},
			path:     "missing/path",
			key:      "api_key",
			fallback: "fallback",
			want:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &BaoProvider{
				kv: &mockKV{
					data: tt.mockData,
					err:  tt.mockErr,
				},
			}

			got := provider.GetSecret(tt.path, tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("GetSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaoProvider_Integration(t *testing.T) {
	// Integration Test: Requires a running OpenBao server.
	if os.Getenv("BAO_ADDR") == "" || os.Getenv("BAO_TOKEN") == "" {
		t.Skip("Skipping integration test: BAO_ADDR or BAO_TOKEN not set")
	}

	provider, err := NewBaoProvider()
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Setup a test secret directly using the client
	testPath := "test-app"
	testKey := "api_key"
	testValue := "super-secret-123"

	data := map[string]interface{}{
		testKey: testValue,
	}

	_, err = provider.client.KVv2("secret").Put(context.Background(), testPath, data)
	if err != nil {
		t.Skipf("Skipping integration test: Failed to put test secret (likely permission or server issue): %v", err)
	}

	got := provider.GetSecret(testPath, testKey, "fallback")
	if got != testValue {
		t.Errorf("GetSecret() = %v, want %v", got, testValue)
	}
}
