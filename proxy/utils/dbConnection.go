package utils

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
		log.Fatalf("❌ %s is not set in environment variables", key)
	}
	return value
}

func InitPostgres(driverName string) *sql.DB {
	host := getRequiredEnv("DB_HOST")
	port := getEnv("DB_PORT", "5432")
	user := getRequiredEnv("DB_USER")
	password := os.Getenv("SERVER_DB_PASSWORD")
	dbname := getRequiredEnv("DB_NAME")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	// If using sqlmock, the connection string might need to be ignored or specific
	// For now, we keep the logic as is.
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		log.Fatalf("❌ Failed to open database connection: %v", err)
	}

	// Critical: test the connection before proceeding
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database (check container/network): %v", err)
	}

	log.Println("✅ Successfully connected to PostgreSQL")

	return db
}

func InitMongo() *mongo.Client {
	mongoURI := getRequiredEnv("MONGO_URI")

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("❌ Failed to connect to MongoDB: %v", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("❌ Failed to ping MongoDB: %v", err)
	}

	log.Println("✅ Connected to MongoDB")
	return client
}
