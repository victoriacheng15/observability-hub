package secrets

import (
	"context"
	"os"
	"testing"
)

func TestBaoProvider_Integration(t *testing.T) {
	// Integration Test: Requires a running OpenBao server.
	//
	// Run manually with:
	// nix-shell --run "export BAO_ADDR='http://127.0.0.1:8200' && export BAO_TOKEN='<root-token>' && cd pkg/secrets && go test -v ."
	//
	// If variables are missing, this test is skipped to prevent build failures in CI/CD.
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
		t.Fatalf("Failed to put test secret: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		key      string
		fallback string
		want     string
	}{
		{
			name:     "Retrieve existing secret",
			path:     testPath,
			key:      testKey,
			fallback: "fallback",
			want:     testValue,
		},
		{
			name:     "Fallback on missing key",
			path:     testPath,
			key:      "wrong_key",
			fallback: "fallback",
			want:     "fallback",
		},
		{
			name:     "Fallback on missing path",
			path:     "missing/path",
			key:      "key",
			fallback: "fallback",
			want:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.GetSecret(tt.path, tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("GetSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}
