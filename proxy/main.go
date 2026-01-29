package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	"db"
	"logger"
	"proxy/utils"
	"secrets"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	// Initialize structured logging first
	logger.Setup(os.Stdout, "proxy")

	godotenv.Load(".env")
	godotenv.Load("../.env")

	var dbPostgres *sql.DB
	var mongoClient *mongo.Client

	// Initialize Secrets Provider
	secretStore, err := secrets.NewBaoProvider()
	if err != nil {
		slog.Error("secret_provider_init_failed", "error", err)
		os.Exit(1)
	}

	dbPostgres = initPostgres("postgres", secretStore)
	mongoClient = initMongo(secretStore)

	// Initialize the reading service
	readingService := &utils.ReadingService{
		DB:          dbPostgres,
		MongoClient: mongoClient,
	}

	// Determine port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	// Register HTTP handlers with logging middleware
	http.HandleFunc("/", utils.WithLogging(utils.HomeHandler))
	http.HandleFunc("/api/health", utils.WithLogging(utils.HealthHandler))
	http.HandleFunc("/api/sync/reading", utils.WithLogging(readingService.SyncReadingHandler))
	http.HandleFunc("/api/webhook/gitops", utils.WithLogging(utils.WebhookHandler))

	slog.Info("🚀 The GO proxy listening on port", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func initPostgres(driverName string, store secrets.SecretStore) *sql.DB {
	database, err := db.ConnectPostgres(driverName, store)
	if err != nil {
		slog.Error("db_connection_failed", "database", "postgres", "error", err)
		os.Exit(1)
	}

	slog.Info("db_connected", "database", "postgres")
	return database
}

func initMongo(store secrets.SecretStore) *mongo.Client {
	client, err := db.ConnectMongo(store)
	if err != nil {
		slog.Error("db_connection_failed", "database", "mongodb", "error", err)
		os.Exit(1)
	}

	slog.Info("db_connected", "database", "mongodb")
	return client
}
