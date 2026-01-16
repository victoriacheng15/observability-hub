package db

import (
	"os"
	"testing"
)

func TestGetMongoURI_MissingEnv(t *testing.T) {
	// Ensure MONGO_URI is unset
	os.Unsetenv("MONGO_URI")

	uri, err := GetMongoURI()
	if err == nil {
		t.Error("Expected error when MONGO_URI is missing, got nil")
	}
	if uri != "" {
		t.Errorf("Expected empty URI when MONGO_URI is missing, got %s", uri)
	}
}

func TestGetMongoURI_Success(t *testing.T) {
	expected := "mongodb://user:pass@localhost:27017"
	os.Setenv("MONGO_URI", expected)
	defer os.Unsetenv("MONGO_URI")

	uri, err := GetMongoURI()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if uri != expected {
		t.Errorf("Expected URI %q, got %q", expected, uri)
	}
}

func TestGetPostgresDSN_Defaults(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("SERVER_DB_PASSWORD", "password")
	os.Setenv("DB_NAME", "mydb")
	// DB_PORT defaults to 5432

	dsn, err := GetPostgresDSN()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "host=localhost port=5432 user=postgres password=password dbname=mydb sslmode=disable timezone=UTC"
	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}
