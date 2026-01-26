package utils

import (
	"database/sql"
	"log/slog"
	"os"

	"db"
	"secrets"

	"go.mongodb.org/mongo-driver/mongo"
)

func InitPostgres(driverName string, store secrets.SecretStore) *sql.DB {
	database, err := db.ConnectPostgres(driverName, store)
	if err != nil {
		slog.Error("db_connection_failed", "database", "postgres", "error", err)
		os.Exit(1)
	}

	slog.Info("db_connected", "database", "postgres")
	return database
}

func InitMongo(store secrets.SecretStore) *mongo.Client {
	client, err := db.ConnectMongo(store)
	if err != nil {
		slog.Error("db_connection_failed", "database", "mongodb", "error", err)
		os.Exit(1)
	}

	slog.Info("db_connected", "database", "mongodb")
	return client
}
