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

func InitPostgres() *sql.DB {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("SERVER_DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	if host == "" {
		log.Fatal("❌ DB_HOST is not set in environment variables")
	}

	if user == "" {
		log.Fatal("❌ DB_USER is not set in environment variables")
	}

	if dbname == "" {
		log.Fatal("❌ DB_NAME is not set in environment variables")
	}

	if port == "" {
		port = "5432"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", connStr)
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
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("❌ MONGO_URI is not set in environment variables")
	}

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
