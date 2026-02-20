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

// Wrapper returns a PostgresWrapper instance using the mock DB.
func (m *MockDB) Wrapper() *PostgresWrapper {
	return &PostgresWrapper{DB: m.DB}
}

// AnyArg returns a placeholder that matches any argument in a SQL query.
func (m *MockDB) AnyArg() sqlmock.Argument {
	return sqlmock.AnyArg()
}

// NewResult returns a driver.Result that can be used to simulate an Exec result.
func (m *MockDB) NewResult(lastInsertID, rowsAffected int64) sql.Result {
	return sqlmock.NewResult(lastInsertID, rowsAffected)
}

// NewRows returns a new sqlmock.Rows instance.
func (m *MockDB) NewRows(columns []string) *sqlmock.Rows {
	return sqlmock.NewRows(columns)
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
