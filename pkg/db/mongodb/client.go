// Package mongodb provides a pure, OTel-instrumented wrapper for MongoDB.
// Supported Operations:
// - Connection: NewMongoStore (via OpenBao/Env)
// - Read: Find (Generic helper for multiple documents)
// - Update: UpdateByID (Standardized helper for single document update)
package mongodb

import (
	"context"
	"fmt"
	"os"
	"strings"
	"telemetry"
	"time"

	"secrets"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	mongoConnect = mongo.Connect
	tracer       = telemetry.GetTracer("db/mongodb")
)

// MongoStore provides a standardized, OTel-instrumented wrapper around mongo.Client.
type MongoStore struct {
	Client *mongo.Client
}

// NewMongoStore establishes a connection to MongoDB and returns a purely generic MongoStore wrapper.
func NewMongoStore(store secrets.SecretStore) (*MongoStore, error) {
	client, err := ConnectMongo(store)
	if err != nil {
		return nil, err
	}
	return &MongoStore{Client: client}, nil
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

	span.SetAttributes(
		telemetry.StringAttribute("db.system", "mongodb"),
		telemetry.StringAttribute("db.name", database),
		telemetry.StringAttribute("db.collection", collection),
		telemetry.IntAttribute("db.query.limit", int(limit)),
	)

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
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

	span.SetAttributes(
		telemetry.StringAttribute("db.system", "mongodb"),
		telemetry.StringAttribute("db.name", database),
		telemetry.StringAttribute("db.collection", collection),
		telemetry.StringAttribute("db.mongodb.id", id),
	)

	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.CodeError, "invalid_object_id")
		return fmt.Errorf("invalid object id: %w", err)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongoConnect(options.Client().ApplyURI(uri))
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
