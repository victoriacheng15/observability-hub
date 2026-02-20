// Package postgres provides a pure, OTel-instrumented wrapper for PostgreSQL.
// Supported Operations:
// - Connection: ConnectPostgres (via OpenBao/Env)
// - Write: Exec (Generic helper for INSERT, UPDATE, DELETE)
// - Read: Query, QueryRow (Generic helpers for SELECT)
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"secrets"
	"telemetry"

	"github.com/lib/pq"
)

var (
	sqlOpen = sql.Open
	tracer  = telemetry.GetTracer("db/postgres")
)

// PostgresWrapper provides a standardized, OTel-instrumented wrapper around sql.DB.
type PostgresWrapper struct {
	DB *sql.DB
}

// ConnectPostgres establishes a connection to PostgreSQL and returns a PostgresWrapper.
func ConnectPostgres(driverName string, store secrets.SecretStore) (*PostgresWrapper, error) {
	dsn, err := GetPostgresDSN(store)
	if err != nil {
		return nil, err
	}

	db, err := sqlOpen(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresWrapper{DB: db}, nil
}

// Array returns a wrapper for a slice that can be used as a PostgreSQL array in queries.
// This allows services to use arrays without importing github.com/lib/pq.
func (w *PostgresWrapper) Array(v any) any {
	return pq.Array(v)
}

// Exec executes a query without returning any rows, with automatic OTel instrumentation.
func (w *PostgresWrapper) Exec(ctx context.Context, opName, query string, args ...any) (sql.Result, error) {
	ctx, span := tracer.Start(ctx, opName)
	defer span.End()

	span.SetAttributes(
		telemetry.StringAttribute("db.system", "postgresql"),
		telemetry.StringAttribute("db.statement", query),
	)

	res, err := w.DB.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.CodeError, err.Error())
		return nil, err
	}

	return res, nil
}

// QueryRow executes a query that is expected to return at most one row.
func (w *PostgresWrapper) QueryRow(ctx context.Context, opName, query string, args ...any) *sql.Row {
	ctx, span := tracer.Start(ctx, opName)
	defer span.End()

	span.SetAttributes(
		telemetry.StringAttribute("db.system", "postgresql"),
		telemetry.StringAttribute("db.statement", query),
	)

	return w.DB.QueryRowContext(ctx, query, args...)
}

// Query executes a query that returns rows, typically a SELECT.
func (w *PostgresWrapper) Query(ctx context.Context, opName, query string, args ...any) (*sql.Rows, error) {
	ctx, span := tracer.Start(ctx, opName)
	defer span.End()

	span.SetAttributes(
		telemetry.StringAttribute("db.system", "postgresql"),
		telemetry.StringAttribute("db.statement", query),
	)

	rows, err := w.DB.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.CodeError, err.Error())
		return nil, err
	}

	return rows, nil
}

// GetPostgresDSN constructs the DSN using the SecretStore.
func GetPostgresDSN(store secrets.SecretStore) (string, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}

	const secretPath = "observability-hub/postgres"

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
