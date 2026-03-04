package mongodb

import "context"

// MockMongoStore provides a flexible mock implementation of MongoStore for testing.
type MockMongoStore struct {
	FindFn       func(ctx context.Context, opName, collection string, filter any, results any, limit int64) error
	UpdateByIDFn func(ctx context.Context, opName, collection string, id string, update any) error
	CloseFn      func(ctx context.Context) error
}

// Find executes the mocked FindFn if provided.
func (m *MockMongoStore) Find(ctx context.Context, opName, collection string, filter any, results any, limit int64) error {
	if m.FindFn != nil {
		return m.FindFn(ctx, opName, collection, filter, results, limit)
	}
	return nil
}

// UpdateByID executes the mocked UpdateByIDFn if provided.
func (m *MockMongoStore) UpdateByID(ctx context.Context, opName, collection string, id string, update any) error {
	if m.UpdateByIDFn != nil {
		return m.UpdateByIDFn(ctx, opName, collection, id, update)
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
