package postgres

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockDB provides a wrapper around sql.DB and sqlmock.Sqlmock to simplify testing.
type MockDB struct {
	Mock sqlmock.Sqlmock
	DB   *sql.DB
}

// NewMockDB initializes a new MockDB instance and returns a cleanup function.
func NewMockDB(t *testing.T) (*MockDB, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return &MockDB{Mock: mock, DB: db}, func() { db.Close() }
}

// ExpectTableCreation sets up an expectation for a CREATE TABLE IF NOT EXISTS statement.
func (m *MockDB) ExpectTableCreation(tableName string) {
	m.Mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + tableName).
		WillReturnResult(sqlmock.NewResult(0, 0))
}

// ExpectHypertableCreation sets up an expectation for a SELECT create_hypertable statement.
func (m *MockDB) ExpectHypertableCreation(tableName string) {
	m.Mock.ExpectExec("SELECT create_hypertable").
		WillReturnResult(sqlmock.NewResult(0, 0))
}
