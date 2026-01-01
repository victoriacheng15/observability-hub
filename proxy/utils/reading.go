package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

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
		log.Printf("❌ ETL_ERROR: Failed to create reading_analytics table: %v", err)
		http.Error(w, "Failed to ensure database schema", 500)
		return
	}

	coll := s.getMongoCollection()
	cursor, err := s.fetchIngestedDocuments(ctx, coll)
	if err != nil {
		log.Printf("❌ ETL_ERROR: Failed to query Mongo: %v", err)
		http.Error(w, "Failed to query Mongo", 500)
		return
	}
	defer cursor.Close(ctx)

	processedCount := s.processDocuments(ctx, cursor, coll)

	log.Printf("✅ ETL_SUCCESS: Processed batch of %d documents", processedCount)
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
		event_type TEXT,
		payload JSONB,
		created_at TIMESTAMP DEFAULT NOW()
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
	opts := options.Find().SetLimit(10)
	return coll.Find(ctx, filter, opts)
}

func (s *ReadingService) processDocuments(ctx context.Context, cursor *mongo.Cursor, coll *mongo.Collection) int {
	processedCount := 0

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("⚠️ ETL_WARN: Failed to decode document: %v", err)
			continue
		}

		objID, ok := doc["_id"].(primitive.ObjectID)
		if !ok {
			log.Printf("⚠️ ETL_WARN: Document missing ObjectID")
			continue
		}

		if err := s.insertIntoPostgres(doc, objID); err != nil {
			log.Printf("❌ ETL_ERROR: Failed to insert into Postgres (ID: %s): %v", objID.Hex(), err)
			continue
		}

		if err := s.updateMongoStatus(ctx, coll, objID); err != nil {
			log.Printf("⚠️ ETL_WARN: Failed to update Mongo status (ID: %s): %v", objID.Hex(), err)
		} else {
			processedCount++
		}
	}

	return processedCount
}

func (s *ReadingService) insertIntoPostgres(doc bson.M, objID primitive.ObjectID) error {
	eventType, _ := doc["event_type"].(string)
	delete(doc, "status")
	jsonData, _ := json.Marshal(doc)

	_, err := s.DB.Exec(
		`INSERT INTO reading_analytics (mongo_id, event_type, payload, created_at) 
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (mongo_id) DO NOTHING`,
		objID.Hex(), eventType, jsonData,
	)
	return err
}

func (s *ReadingService) updateMongoStatus(ctx context.Context, coll *mongo.Collection, objID primitive.ObjectID) error {
	update := bson.M{"$set": bson.M{"status": "processed"}}
	_, err := coll.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}
