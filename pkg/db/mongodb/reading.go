package mongodb

import (
	"context"
	"fmt"
	"telemetry"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var readingTracer = telemetry.GetTracer("db/mongodb/reading")

// ReadingDocument represents a normalized reading event for external use.
type ReadingDocument struct {
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Type      string                 `json:"event_type"`
	Timestamp interface{}            `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Meta      map[string]interface{} `json:"meta"`
}

// StoreAPI defines the interface for MongoDB operations to facilitate testing.
type StoreAPI interface {
	FetchIngestedArticles(ctx context.Context, limit int64) ([]ReadingDocument, error)
	MarkArticleAsProcessed(ctx context.Context, id string) error
}

// FetchIngestedArticles retrieves documents marked as 'ingested' from MongoDB.
func (s *MongoStore) FetchIngestedArticles(ctx context.Context, limit int64) ([]ReadingDocument, error) {
	ctx, span := readingTracer.Start(ctx, "db.mongodb.fetch_ingested_articles")
	defer span.End()
	span.SetAttributes(
		telemetry.StringAttribute("db.system", "mongodb"),
		telemetry.IntAttribute("db.query.limit", int(limit)),
	)

	coll := s.Client.Database("reading-analytics").Collection("articles")
	filter := bson.M{"status": "ingested"}
	opts := options.Find().SetLimit(limit)

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		span.SetStatus(telemetry.CodeError, "find_failed")
		span.SetAttributes(
			telemetry.BoolAttribute("error", true),
			telemetry.StringAttribute("error.message", err.Error()),
		)
		return nil, fmt.Errorf("failed to find articles: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []ReadingDocument
	for cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			continue
		}

		objID, ok := raw["_id"].(bson.ObjectID)
		if !ok {
			continue
		}

		source, _ := raw["source"].(string)
		eventType, _ := raw["event_type"].(string)

		docs = append(docs, ReadingDocument{
			ID:        objID.Hex(),
			Source:    source,
			Type:      eventType,
			Timestamp: raw["timestamp"],
			Payload:   convertToMap(raw["payload"]),
			Meta:      convertToMap(raw["meta"]),
		})
	}
	span.SetAttributes(telemetry.IntAttribute("db.documents.count", len(docs)))
	return docs, nil
}

// MarkArticleAsProcessed updates the status of an article in MongoDB.
func (s *MongoStore) MarkArticleAsProcessed(ctx context.Context, id string) error {
	ctx, span := readingTracer.Start(ctx, "db.mongodb.mark_article_processed")
	defer span.End()
	span.SetAttributes(
		telemetry.StringAttribute("db.system", "mongodb"),
		telemetry.StringAttribute("doc.id", id),
	)

	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		span.SetStatus(telemetry.CodeError, "invalid_object_id")
		span.SetAttributes(
			telemetry.BoolAttribute("error", true),
			telemetry.StringAttribute("error.message", err.Error()),
		)
		return fmt.Errorf("invalid object id: %w", err)
	}

	coll := s.Client.Database("reading-analytics").Collection("articles")
	filter := bson.M{"_id": objID}
	update := bson.M{"$set": bson.M{"status": "processed"}}

	_, err = coll.UpdateOne(ctx, filter, update)
	if err != nil {
		span.SetStatus(telemetry.CodeError, "update_failed")
		span.SetAttributes(
			telemetry.BoolAttribute("error", true),
			telemetry.StringAttribute("error.message", err.Error()),
		)
		return fmt.Errorf("failed to update article status: %w", err)
	}
	return nil
}

func convertToMap(v interface{}) map[string]interface{} {
	if m, ok := v.(bson.M); ok {
		return map[string]interface{}(m)
	}
	if d, ok := v.(bson.D); ok {
		m := make(map[string]interface{})
		for _, e := range d {
			m[e.Key] = e.Value
		}
		return m
	}
	return nil
}
