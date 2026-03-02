package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// simpleMockStore satisfies the secrets.SecretStore interface for testing
type simpleMockStore struct {
	values map[string]string
}

func (m *simpleMockStore) GetSecret(path, key, fallback string) string {
	if val, ok := m.values[key]; ok {
		return val
	}
	return fallback
}

func (m *simpleMockStore) Close() error { return nil }

func TestConnectPostgres(t *testing.T) {
	mock := &simpleMockStore{
		values: map[string]string{
			"host":               "localhost",
			"port":               "5432",
			"user":               "user",
			"dbname":             "db",
			"server_db_password": "pass",
		},
	}

	t.Run("Success", func(t *testing.T) {
		db, mockSQL, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
		mockSQL.ExpectPing()

		oldSqlOpen := sqlOpen
		defer func() { sqlOpen = oldSqlOpen }()
		sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, nil
		}

		wrapper, err := ConnectPostgres("mock-postgres", mock)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if wrapper == nil || wrapper.DB == nil {
			t.Fatal("Expected wrapper instance with DB, got nil")
		}
	})

	t.Run("Open Failure", func(t *testing.T) {
		oldSqlOpen := sqlOpen
		defer func() { sqlOpen = oldSqlOpen }()
		sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, errors.New("open failed")
		}
		_, err := ConnectPostgres("mock-postgres", mock)
		if err == nil {
			t.Error("Expected error for open failure, got nil")
		}
	})

	t.Run("Ping Failure", func(t *testing.T) {
		db, mockSQL, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
		mockSQL.ExpectPing().WillReturnError(errors.New("ping failed"))

		oldSqlOpen := sqlOpen
		defer func() { sqlOpen = oldSqlOpen }()
		sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, nil
		}

		_, err := ConnectPostgres("mock-postgres", mock)
		if err == nil {
			t.Error("Expected error for ping failure, got nil")
		}
	})
}

func TestPostgresWrapper_Operations(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()
	wrapper := mdb.Wrapper()
	ctx := context.Background()

	t.Run("Exec Success", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO test").
			WithArgs(1, "test").
			WillReturnResult(mdb.NewResult(1, 1))

		_, err := wrapper.Exec(ctx, "test-op", "INSERT INTO test VALUES ($1, $2)", 1, "test")
		if err != nil {
			t.Errorf("Exec failed: %v", err)
		}
	})

	t.Run("Exec Failure", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO test").
			WillReturnError(errors.New("exec error"))

		_, err := wrapper.Exec(ctx, "test-op", "INSERT INTO test")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("Query Success", func(t *testing.T) {
		rows := mdb.NewRows([]string{"id", "name"}).AddRow(1, "test")
		mdb.Mock.ExpectQuery("SELECT").WillReturnRows(rows)

		res, err := wrapper.Query(ctx, "test-op", "SELECT * FROM test")
		if err != nil {
			t.Errorf("Query failed: %v", err)
		}
		res.Close()
	})

	t.Run("Query Failure", func(t *testing.T) {
		mdb.Mock.ExpectQuery("SELECT").WillReturnError(errors.New("query error"))

		_, err := wrapper.Query(ctx, "test-op", "SELECT * FROM test")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("QueryRow", func(t *testing.T) {
		mdb.Mock.ExpectQuery("SELECT").WillReturnRows(mdb.NewRows([]string{"id"}).AddRow(1))
		row := wrapper.QueryRow(ctx, "test-op", "SELECT id FROM test")
		var id int
		if err := row.Scan(&id); err != nil {
			t.Errorf("QueryRow Scan failed: %v", err)
		}
	})

	t.Run("Array Helper", func(t *testing.T) {
		arr := wrapper.Array([]string{"a", "b"})
		if arr == nil {
			t.Error("Expected non-nil array wrapper")
		}
	})

	t.Run("AnyArg", func(t *testing.T) {
		if mdb.AnyArg() == nil {
			t.Error("Expected non-nil AnyArg")
		}
	})
}

func TestGetPostgresDSN(t *testing.T) {
	t.Run("DATABASE_URL Env", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://user:pass@host/db")
		defer os.Unsetenv("DATABASE_URL")
		dsn, _ := GetPostgresDSN(nil)
		if dsn != "postgres://user:pass@host/db" {
			t.Errorf("Expected DATABASE_URL, got %s", dsn)
		}
	})

	t.Run("Missing Credentials", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("SERVER_DB_PASSWORD")
		mock := &simpleMockStore{values: map[string]string{}}
		_, err := GetPostgresDSN(mock)
		if err == nil {
			t.Error("Expected error for missing credentials, got nil")
		}
	})

	t.Run("getEnv Fallback", func(t *testing.T) {
		os.Unsetenv("NON_EXISTENT_VAR")
		val := getEnv("NON_EXISTENT_VAR", "fallback")
		if val != "fallback" {
			t.Errorf("Expected fallback, got %s", val)
		}
	})
}

func TestMockDB_Helpers(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	t.Run("ExpectTableCreation", func(t *testing.T) {
		mdb.ExpectTableCreation("test_table")
	})

	t.Run("ExpectHypertableCreation", func(t *testing.T) {
		mdb.ExpectHypertableCreation("test_table")
	})
}
