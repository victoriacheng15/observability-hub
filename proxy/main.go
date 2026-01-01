package main

import (
	"log"
	"net/http"
	"os"

	"proxy/utils"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load(".env")
	godotenv.Load("../.env")

	db := utils.InitPostgres("postgres")
	mongoClient := utils.InitMongo()

	// Initialize the reading service
	readingService := &utils.ReadingService{
		DB:          db,
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

	log.Printf("🚀 The GO proxy listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
