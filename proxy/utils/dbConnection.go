package utils

import (
	"database/sql"
	"log/slog"
	"os"

	"db"

	"go.mongodb.org/mongo-driver/mongo"
)

func InitPostgres(driverName string) *sql.DB {
	database, err := db.ConnectPostgres(driverName)
	if err != nil {
		slog.Error("db_connection_failed", "database", "postgres", "error", err)
		os.Exit(1)
	}

	slog.Info("db_connected", "database", "postgres")
	return database
}

func InitMongo() *mongo.Client {
	client, err := db.ConnectMongo()
	if err != nil {
		slog.Error("db_connection_failed", "database", "mongodb", "error", err)
		os.Exit(1)
	}

	slog.Info("db_connected", "database", "mongodb")
	return client
}
