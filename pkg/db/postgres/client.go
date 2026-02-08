package postgres

import (
	"database/sql"
	"fmt"
	"os"

	"secrets"

	_ "github.com/lib/pq"
)

// Internal variables for testing
var (
	sqlOpen = sql.Open
)

// ConnectPostgres establishes a connection to PostgreSQL and verifies it with a Ping.
// It accepts a SecretStore to retrieve credentials from OpenBao with env fallbacks.
func ConnectPostgres(driverName string, store secrets.SecretStore) (*sql.DB, error) {
	dsn, err := GetPostgresDSN(store)
	if err != nil {
		return nil, err
	}

	db, err := sqlOpen(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}

// GetPostgresDSN constructs the DSN using the SecretStore.
func GetPostgresDSN(store secrets.SecretStore) (string, error) {
	// 1. Priority: DATABASE_URL (for local dev/testing override)
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}

	// Path relative to the KV mount (e.g., 'secret').
	const secretPath = "observability-hub/postgres"

	// Retrieve values with fallbacks to environment variables or defaults
	host := store.GetSecret(secretPath, "host", getEnv("DB_HOST", "localhost"))
	port := store.GetSecret(secretPath, "port", getEnv("DB_PORT", "30432"))
	user := store.GetSecret(secretPath, "user", getEnv("DB_USER", "server"))
	dbname := store.GetSecret(secretPath, "dbname", getEnv("DB_NAME", "homelab"))
	password := store.GetSecret(secretPath, "server_db_password", os.Getenv("SERVER_DB_PASSWORD"))

	if host == "" || user == "" || dbname == "" || password == "" {
		return "", fmt.Errorf("missing required database credentials (host, user, dbname, or password)")
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC",
		host, port, user, password, dbname,
	), nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
