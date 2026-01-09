package db

import (
	"fmt"
	"os"
	"strings"
)

// GetPostgresDSN returns the formatted connection string based on environment variables.
// It prioritizes DATABASE_URL if set. Otherwise, it constructs the DSN from:
// DB_HOST, DB_PORT, DB_USER, SERVER_DB_PASSWORD, DB_NAME.
// It enforces critical defaults like timezone=UTC and sslmode=disable.
func GetPostgresDSN() (string, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		// If DATABASE_URL is provided, we assume it's correctly formatted.
		// We could append defaults, but usually DATABASE_URL is self-contained.
		return dsn, nil
	}

	host := getEnv("DB_HOST", "")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "")
	password := os.Getenv("SERVER_DB_PASSWORD")
	dbname := getEnv("DB_NAME", "")

	if host == "" || user == "" || dbname == "" || password == "" {
		var missing []string
		if host == "" {
			missing = append(missing, "DB_HOST")
		}
		if user == "" {
			missing = append(missing, "DB_USER")
		}
		if dbname == "" {
			missing = append(missing, "DB_NAME")
		}
		if password == "" {
			missing = append(missing, "SERVER_DB_PASSWORD")
		}
		return "", fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
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
