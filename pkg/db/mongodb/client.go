package mongodb

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"secrets"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Internal variables for testing
var (
	mongoConnect = mongo.Connect
)

// MongoStore wraps the MongoDB client to provide a high-level API.
type MongoStore struct {
	Client *mongo.Client
}

// NewMongoStore establishes a connection to MongoDB and returns a MongoStore.
func NewMongoStore(store secrets.SecretStore) (*MongoStore, error) {
	client, err := ConnectMongo(store)
	if err != nil {
		return nil, err
	}
	return &MongoStore{Client: client}, nil
}

// ConnectMongo establishes a connection to MongoDB.
func ConnectMongo(store secrets.SecretStore) (*mongo.Client, error) {
	uri, err := GetMongoURI(store)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongoConnect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return client, nil
}

func GetMongoURI(store secrets.SecretStore) (string, error) {
	const secretPath = "observability-hub/mongo"

	// Fetch from OpenBao with fallback to legacy MONGO_URI env var
	uri := strings.TrimSpace(store.GetSecret(secretPath, "uri", os.Getenv("MONGO_URI")))

	if uri == "" {
		return "", fmt.Errorf("missing required environment variable: MONGO_URI")
	}
	return uri, nil
}
