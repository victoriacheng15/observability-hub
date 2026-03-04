package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate structure: tmpDir/.env (root) and tmpDir/services/app/.env (local)
	appDir := filepath.Join(tmpDir, "services", "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	rootEnvPath := filepath.Join(tmpDir, ".env")
	localEnvPath := filepath.Join(appDir, ".env")

	tests := []struct {
		name         string
		rootContent  string
		localContent string
		wantVars     map[string]string
	}{
		{
			name:         "Load from both",
			rootContent:  "ROOT_VAR=root_val\nSHARED_VAR=root_shared",
			localContent: "LOCAL_VAR=local_val\nSHARED_VAR=local_shared",
			wantVars: map[string]string{
				"ROOT_VAR":   "root_val",
				"LOCAL_VAR":  "local_val",
				"SHARED_VAR": "local_shared", // Local loaded first, godotenv doesn't overwrite
			},
		},
		{
			name:         "Only root exists",
			rootContent:  "ONLY_ROOT=true",
			localContent: "",
			wantVars: map[string]string{
				"ONLY_ROOT": "true",
			},
		},
		{
			name:         "Only local exists",
			rootContent:  "",
			localContent: "ONLY_LOCAL=true",
			wantVars: map[string]string{
				"ONLY_LOCAL": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up previous test state
			os.Remove(rootEnvPath)
			os.Remove(localEnvPath)
			for k := range tt.wantVars {
				os.Unsetenv(k)
			}

			// Write files if content provided
			if tt.rootContent != "" {
				os.WriteFile(rootEnvPath, []byte(tt.rootContent), 0644)
			}
			if tt.localContent != "" {
				os.WriteFile(localEnvPath, []byte(tt.localContent), 0644)
			}

			// Change to app directory
			originalWD, _ := os.Getwd()
			if err := os.Chdir(appDir); err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}
			defer os.Chdir(originalWD)

			// Run Load
			Load()

			// Verify
			for k, want := range tt.wantVars {
				if got := os.Getenv(k); got != want {
					t.Errorf("Variable %s: got %q, want %q", k, got, want)
				}
			}
		})
	}
}
