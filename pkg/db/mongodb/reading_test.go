package mongodb

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoStore_Reading_Mock(t *testing.T) {
	docID := "507f1f77bcf86cd799439011"

	t.Run("FetchArticles", func(t *testing.T) {
		m := &MockMongoStore{
			FetchFn: func(ctx context.Context, limit int64) ([]ReadingDocument, error) {
				return []ReadingDocument{{ID: docID}}, nil
			},
		}
		docs, _ := m.FetchIngestedArticles(context.Background(), 10)
		if len(docs) != 1 || docs[0].ID != docID {
			t.Errorf("mock failed to return expected document")
		}
	})

	t.Run("MarkProcessed", func(t *testing.T) {
		called := false
		m := &MockMongoStore{
			MarkFn: func(ctx context.Context, id string) error {
				called = true
				return nil
			},
		}
		_ = m.MarkArticleAsProcessed(context.Background(), docID)
		if !called {
			t.Error("mock MarkFn was not called")
		}
	})
}

func TestConvertToMap(t *testing.T) {
	t.Run("bson.M", func(t *testing.T) {
		m := bson.M{"k": "v"}
		res := convertToMap(m)
		if res["k"] != "v" {
			t.Errorf("expected v, got %v", res["k"])
		}
	})

	t.Run("bson.D", func(t *testing.T) {
		d := bson.D{{Key: "k", Value: "v"}}
		res := convertToMap(d)
		if res["k"] != "v" {
			t.Errorf("expected v, got %v", res["k"])
		}
	})

	t.Run("nil", func(t *testing.T) {
		if convertToMap(nil) != nil {
			t.Error("expected nil for nil input")
		}
	})
}
