package main

import (
	"log/slog"
	"net/http"
	"os"

	"logger"
	"proxy/utils"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Initialize structured logging first
	logger.Setup("proxy")

	godotenv.Load(".env")
	godotenv.Load("../.env")

	dbPostgres := utils.InitPostgres("postgres")
	mongoClient := utils.InitMongo()

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
	http.HandleFunc("/api/reading", utils.WithLogging(readingService.ReadingHandler))
	http.HandleFunc("/api/sync/reading", utils.WithLogging(readingService.SyncReadingHandler))

	slog.Info("🚀 The GO proxy listening on port", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
