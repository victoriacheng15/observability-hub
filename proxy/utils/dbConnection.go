package utils

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"db"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("env_var_missing", "key", key)
		os.Exit(1)
	}
	return value
}

func InitPostgres(driverName string) *sql.DB {
	connStr, err := db.GetPostgresDSN()
	if err != nil {
		slog.Error("db_config_failed", "error", err)
		os.Exit(1)
	}

	// If using sqlmock, the connection string might need to be ignored or specific
	// For now, we keep the logic as is.
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		slog.Error("db_connection_failed", "database", "postgres", "error", err)
		os.Exit(1)
	}

	// Critical: test the connection before proceeding
	if err := db.Ping(); err != nil {
		slog.Error("db_ping_failed", "database", "postgres", "error", err)
		os.Exit(1)
	}

	slog.Info("db_connected", "database", "postgres")

	return db
}

func InitMongo() *mongo.Client {
	mongoURI := getRequiredEnv("MONGO_URI")

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		slog.Error("db_connection_failed", "database", "mongodb", "error", err)
		os.Exit(1)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		slog.Error("db_ping_failed", "database", "mongodb", "error", err)
		os.Exit(1)
	}

	slog.Info("db_connected", "database", "mongodb")
	return client
}
