package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ReadingService struct {
	DB          *sql.DB
	MongoClient *mongo.Client
}

func (s *ReadingService) ReadingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"placeholder": "reading placeholder still"})
}

func (s *ReadingService) SyncReadingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	if err := s.ensureReadingAnalyticsTable(); err != nil {
		slog.Error("ETL_ERROR: Failed to create reading_analytics table", "error", err)
		http.Error(w, "Failed to ensure database schema", 500)
		return
	}

	coll := s.getMongoCollection()
	cursor, err := s.fetchIngestedDocuments(ctx, coll)
	if err != nil {
		slog.Error("ETL_ERROR: Failed to query Mongo", "error", err)
		http.Error(w, "Failed to query Mongo", 500)
		return
	}
	defer cursor.Close(ctx)

	processedCount := s.processDocuments(ctx, cursor, coll)

	slog.Info("ETL_SUCCESS: Processed batch", "count", processedCount)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"processed_count": processedCount,
	})
}

func (s *ReadingService) ensureReadingAnalyticsTable() error {
	_, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS reading_analytics (
		id SERIAL PRIMARY KEY,
		mongo_id TEXT UNIQUE NOT NULL,
		event_timestamp TIMESTAMPTZ,
		source TEXT,
		event_type TEXT,
		payload JSONB,
		meta JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	return err
}

func (s *ReadingService) getMongoCollection() *mongo.Collection {
	dbName := os.Getenv("MONGO_DB_NAME")
	collection := os.Getenv("MONGO_COLLECTION")
	return s.MongoClient.Database(dbName).Collection(collection)
}

func (s *ReadingService) fetchIngestedDocuments(ctx context.Context, coll *mongo.Collection) (*mongo.Cursor, error) {
	filter := bson.M{"status": "ingested"}

	batchSize := 100 // Default
	if envSize := os.Getenv("BATCH_SIZE"); envSize != "" {
		if val, err := strconv.Atoi(envSize); err == nil && val > 0 {
			batchSize = val
		}
	}

	opts := options.Find().SetLimit(int64(batchSize))
	return coll.Find(ctx, filter, opts)
}

func (s *ReadingService) processDocuments(ctx context.Context, cursor *mongo.Cursor, coll *mongo.Collection) int {
	processedCount := 0

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			slog.Warn("ETL_WARN: Failed to decode document", "error", err)
			continue
		}

		objID, ok := doc["_id"].(primitive.ObjectID)
		if !ok {
			slog.Warn("ETL_WARN: Document missing ObjectID")
			continue
		}

		if err := s.insertIntoPostgres(doc, objID); err != nil {
			slog.Error("ETL_ERROR: Failed to insert into Postgres", "id", objID.Hex(), "error", err)
			continue
		}

		if err := s.updateMongoStatus(ctx, coll, objID); err != nil {
			slog.Warn("ETL_WARN: Failed to update Mongo status", "id", objID.Hex(), "error", err)
		} else {
			processedCount++
		}
	}

	return processedCount
}

func (s *ReadingService) insertIntoPostgres(doc bson.M, objID primitive.ObjectID) error {
	eventType, _ := doc["event_type"].(string)
	source, _ := doc["source"].(string)
	timestamp := doc["timestamp"]

	payloadJSON, _ := json.Marshal(doc["payload"])
	metaJSON, _ := json.Marshal(doc["meta"])

	_, err := s.DB.Exec(
		`INSERT INTO reading_analytics (mongo_id, event_timestamp, source, event_type, payload, meta, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (mongo_id) DO NOTHING`,
		objID.Hex(), timestamp, source, eventType, payloadJSON, metaJSON,
	)
	return err
}

func (s *ReadingService) updateMongoStatus(ctx context.Context, coll *mongo.Collection, objID primitive.ObjectID) error {
	update := bson.M{"$set": bson.M{"status": "processed"}}
	_, err := coll.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}
