// Package mongodb provides a pure, OTel-instrumented wrapper for MongoDB.
package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"observability-hub/internal/telemetry"
	"os"
	"strings"
	"time"

	"observability-hub/internal/secrets"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	mongoConnect = mongo.Connect
	tracer       = telemetry.GetTracer("db/mongodb")
)

// IMongoStore defines the interface for our MongoDB operations.
type IMongoStore interface {
	Find(ctx context.Context, opName, database, collection string, filter any, results any, limit int64) error
	UpdateByID(ctx context.Context, opName, database, collection string, id string, update any) error
	Close(ctx context.Context) error
}

// MongoStore provides a standardized, OTel-instrumented wrapper around mongo.Client.
type MongoStore struct {
	Client *mongo.Client
	user   string
}

// NewMongoStore establishes a connection to MongoDB and returns a purely generic MongoStore wrapper.
func NewMongoStore(store secrets.SecretStore) (*MongoStore, error) {
	uri, err := GetMongoURI(store)
	if err != nil {
		return nil, err
	}

	client, err := ConnectMongoWithURI(uri)
	if err != nil {
		return nil, err
	}

	// Basic user extraction from URI (e.g., mongodb://user:pass@host)
	user := "unknown"
	if strings.HasPrefix(uri, "mongodb://") {
		parts := strings.SplitN(strings.TrimPrefix(uri, "mongodb://"), "@", 2)
		if len(parts) > 1 {
			userPass := strings.SplitN(parts[0], ":", 2)
			user = userPass[0]
		}
	}

	return &MongoStore{Client: client, user: user}, nil
}

// Close disconnects the MongoDB client.
func (w *MongoStore) Close(ctx context.Context) error {
	if w.Client != nil {
		return w.Client.Disconnect(ctx)
	}
	return nil
}

// Find retrieves multiple documents from a database/collection with a limit and OTel instrumentation.
func (w *MongoStore) Find(ctx context.Context, opName, database, collection string, filter any, results any, limit int64) error {
	ctx, span := tracer.Start(ctx, opName)
	defer span.End()

	stmt, _ := json.Marshal(filter)

	span.SetAttributes(
		telemetry.StringAttribute("db.system", "mongodb"),
		telemetry.StringAttribute("db.name", database),
		telemetry.StringAttribute("db.collection", collection),
		telemetry.StringAttribute("db.statement", string(stmt)),
		telemetry.StringAttribute("db.user", w.user),
		telemetry.IntAttribute("db.query.limit", int(limit)),
	)

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
	}

	if w.Client == nil {
		return fmt.Errorf("mongo client is nil")
	}

	coll := w.Client.Database(database).Collection(collection)
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.CodeError, err.Error())
		return err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, results); err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.CodeError, err.Error())
		return err
	}

	return nil
}

// UpdateByID updates a single document by its hexadecimal string ID in a specific database/collection.
func (w *MongoStore) UpdateByID(ctx context.Context, opName, database, collection string, id string, update any) error {
	ctx, span := tracer.Start(ctx, opName)
	defer span.End()

	stmt, _ := json.Marshal(update)

	span.SetAttributes(
		telemetry.StringAttribute("db.system", "mongodb"),
		telemetry.StringAttribute("db.name", database),
		telemetry.StringAttribute("db.collection", collection),
		telemetry.StringAttribute("db.mongodb.id", id),
		telemetry.StringAttribute("db.statement", string(stmt)),
		telemetry.StringAttribute("db.user", w.user),
	)

	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.CodeError, "invalid_object_id")
		return fmt.Errorf("invalid object id: %w", err)
	}

	if w.Client == nil {
		return fmt.Errorf("mongo client is nil")
	}

	coll := w.Client.Database(database).Collection(collection)
	_, err = coll.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.CodeError, err.Error())
		return err
	}

	return nil
}

// ConnectMongo establishes a connection to MongoDB.
func ConnectMongo(store secrets.SecretStore) (*mongo.Client, error) {
	uri, err := GetMongoURI(store)
	if err != nil {
		return nil, err
	}
	return ConnectMongoWithURI(uri)
}

// ConnectMongoWithURI establishes a connection using a raw URI.
func ConnectMongoWithURI(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongoConnect(options.Client().ApplyURI(uri).SetBSONOptions(&options.BSONOptions{
		ObjectIDAsHexString: true,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return client, nil
}

// GetMongoURI retrieves the MongoDB URI from the SecretStore.
func GetMongoURI(store secrets.SecretStore) (string, error) {
	const secretPath = "observability-hub/mongo"
	uri := strings.TrimSpace(store.GetSecret(secretPath, "uri", os.Getenv("MONGO_URI")))

	if uri == "" {
		return "", fmt.Errorf("missing required environment variable: MONGO_URI")
	}
	return uri, nil
}
