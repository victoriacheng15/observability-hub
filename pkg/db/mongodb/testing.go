package mongodb

import "context"

// MockMongoStore provides a flexible mock implementation of StoreAPI for testing.
type MockMongoStore struct {
	FetchFn func(ctx context.Context, limit int64) ([]ReadingDocument, error)
	MarkFn  func(ctx context.Context, id string) error
	CloseFn func(ctx context.Context) error
}

// FetchIngestedArticles executes the mocked FetchFn if provided.
func (m *MockMongoStore) FetchIngestedArticles(ctx context.Context, limit int64) ([]ReadingDocument, error) {
	if m.FetchFn != nil {
		return m.FetchFn(ctx, limit)
	}
	return nil, nil
}

// MarkArticleAsProcessed executes the mocked MarkFn if provided.
func (m *MockMongoStore) MarkArticleAsProcessed(ctx context.Context, id string) error {
	if m.MarkFn != nil {
		return m.MarkFn(ctx, id)
	}
	return nil
}

// Close executes the mocked CloseFn if provided.
func (m *MockMongoStore) Close(ctx context.Context) error {
	if m.CloseFn != nil {
		return m.CloseFn(ctx)
	}
	return nil
}
