package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectPostgres establishes a connection to PostgreSQL and verifies it with a Ping.
// It returns a standard *sql.DB interface.
func ConnectPostgres(driverName string) (*sql.DB, error) {
	dsn, err := GetPostgresDSN()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}

// ConnectMongo establishes a connection to MongoDB Atlas and verifies it with a Ping.
func ConnectMongo() (*mongo.Client, error) {
	uri, err := GetMongoURI()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Test connection
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer pingCancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return client, nil
}

func GetMongoURI() (string, error) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return "", fmt.Errorf("missing required environment variable: MONGO_URI")
	}
	return uri, nil
}

func GetPostgresDSN() (string, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}

	host := getEnv("DB_HOST", "")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "")
	password := os.Getenv("SERVER_DB_PASSWORD")
	dbname := getEnv("DB_NAME", "")

	if host == "" || user == "" || dbname == "" || password == "" {
		var missing []string
		if host == "" {
			missing = append(missing, "DB_HOST")
		}
		if user == "" {
			missing = append(missing, "DB_USER")
		}
		if dbname == "" {
			missing = append(missing, "DB_NAME")
		}
		if password == "" {
			missing = append(missing, "SERVER_DB_PASSWORD")
		}
		return "", fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
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
