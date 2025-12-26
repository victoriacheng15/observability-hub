package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		val      string
		fallback []string
		expected string
	}{
		{"existing env", "TEST_ENV_VAR", "hello", nil, "hello"},
		{"missing with fallback", "MISSING_VAR", "", []string{"world"}, "world"},
		{"missing without fallback", "MISSING_VAR", "", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.val != "" {
				os.Setenv(tt.key, tt.val)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.fallback...)
			if result != tt.expected {
				t.Errorf("getEnv(%s) = %s, want %s", tt.key, result, tt.expected)
			}
		})
	}
}

func TestGetConnStr_DatabaseURL(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	defer os.Unsetenv("DATABASE_URL")

	expected := "postgres://user:pass@localhost:5432/db"
	if result := getConnStr(); result != expected {
		t.Errorf("getConnStr() = %s, want %s", result, expected)
	}
}

func TestGetConnStr_Parts(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("SERVER_DB_PASSWORD", "secret")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("SERVER_DB_PASSWORD")
	}()

	result := getConnStr()
	if !strings.Contains(result, "host=localhost") ||
		!strings.Contains(result, "user=testuser") ||
		!strings.Contains(result, "dbname=testdb") ||
		!strings.Contains(result, "password=secret") {
		t.Errorf("getConnStr() returned unexpected string: %s", result)
	}
}

func TestGetConnStr_MissingRequired(t *testing.T) {
	// Subprocess test for log.Fatal
	if os.Getenv("BE_CRASHER") == "1" {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("DB_HOST")
		getConnStr()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestGetConnStr_MissingRequired")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
