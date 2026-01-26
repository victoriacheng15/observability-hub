package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	"logger"
	"proxy/utils"
	"secrets"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	// Initialize structured logging first
	logger.Setup("proxy")

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

	dbPostgres = utils.InitPostgres("postgres", secretStore)
	mongoClient = utils.InitMongo(secretStore)

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
