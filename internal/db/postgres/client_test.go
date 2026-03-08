package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestPostgresWrapper_OTelAttributes(t *testing.T) {
	// Setup test exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)
	otel.SetTracerProvider(tp)

	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	// Create wrapper with explicit user and db for testing
	wrapper := &PostgresWrapper{
		DB:   mdb.DB,
		user: "test-user",
		db:   "test-db",
	}
	ctx := context.Background()

	t.Run("Verify Exec Attributes", func(t *testing.T) {
		exporter.Reset()
		mdb.Mock.ExpectExec("INSERT").WillReturnResult(mdb.NewResult(1, 1))

		_, _ = wrapper.Exec(ctx, "test-exec", "INSERT INTO t VALUES (1)")

		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("Expected span, got none")
		}

		attrs := make(map[string]string)
		for _, a := range spans[0].Attributes {
			attrs[string(a.Key)] = a.Value.AsString()
		}

		expected := map[string]string{
			"db.system":    "postgresql",
			"db.statement": "INSERT INTO t VALUES (1)",
			"db.user":      "test-user",
			"db.name":      "test-db",
		}

		for k, v := range expected {
			if attrs[k] != v {
				t.Errorf("Expected attribute %s=%s, got %s", k, v, attrs[k])
			}
		}
	})

	t.Run("Verify Query Attributes", func(t *testing.T) {
		exporter.Reset()
		rows := mdb.NewRows([]string{"id"}).AddRow(1)
		mdb.Mock.ExpectQuery("SELECT").WillReturnRows(rows)

		res, _ := wrapper.Query(ctx, "test-query", "SELECT id FROM t")
		res.Close()

		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("Expected span, got none")
		}

		attrs := make(map[string]string)
		for _, a := range spans[0].Attributes {
			attrs[string(a.Key)] = a.Value.AsString()
		}

		if attrs["db.statement"] != "SELECT id FROM t" {
			t.Errorf("Expected db.statement SELECT id FROM t, got %s", attrs["db.statement"])
		}
		if attrs["db.user"] != "test-user" {
			t.Errorf("Expected db.user test-user, got %s", attrs["db.user"])
		}
	})
}

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
			"dbname":             "observability-hub/internal/db",
			"server_db_password": "pass",
		},
	}

	tests := []struct {
		name    string
		setup   func(mockSQL sqlmock.Sqlmock)
		openErr error
		wantErr bool
	}{
		{
			name: "Success",
			setup: func(mockSQL sqlmock.Sqlmock) {
				mockSQL.ExpectPing()
			},
			wantErr: false,
		},
		{
			name:    "Open Failure",
			openErr: errors.New("open failed"),
			wantErr: true,
		},
		{
			name: "Ping Failure",
			setup: func(mockSQL sqlmock.Sqlmock) {
				mockSQL.ExpectPing().WillReturnError(errors.New("ping failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSQL, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
			if tt.setup != nil {
				tt.setup(mockSQL)
			}

			oldSqlOpen := sqlOpen
			defer func() { sqlOpen = oldSqlOpen }()
			sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
				if tt.openErr != nil {
					return nil, tt.openErr
				}
				return db, nil
			}

			wrapper, err := ConnectPostgres("mock-postgres", mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ConnectPostgres() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if wrapper == nil || wrapper.DB == nil {
					t.Fatal("Expected wrapper instance with DB, got nil")
				}
			}
		})
	}
}

func TestPostgresWrapper_Operations(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()
	wrapper := mdb.Wrapper()
	ctx := context.Background()

	tests := []struct {
		name    string
		testFn  func(t *testing.T)
		wantErr bool
	}{
		{
			name: "Exec Success",
			testFn: func(t *testing.T) {
				mdb.Mock.ExpectExec("INSERT INTO test").
					WithArgs(1, "test").
					WillReturnResult(mdb.NewResult(1, 1))

				_, err := wrapper.Exec(ctx, "test-op", "INSERT INTO test VALUES ($1, $2)", 1, "test")
				if err != nil {
					t.Errorf("Exec failed: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "Exec Failure",
			testFn: func(t *testing.T) {
				mdb.Mock.ExpectExec("INSERT INTO test").
					WillReturnError(errors.New("exec error"))

				_, err := wrapper.Exec(ctx, "test-op", "INSERT INTO test")
				if err == nil {
					t.Error("Expected error, got nil")
				}
			},
			wantErr: true,
		},
		{
			name: "Query Success",
			testFn: func(t *testing.T) {
				rows := mdb.NewRows([]string{"id", "name"}).AddRow(1, "test")
				mdb.Mock.ExpectQuery("SELECT").WillReturnRows(rows)

				res, err := wrapper.Query(ctx, "test-op", "SELECT * FROM test")
				if err != nil {
					t.Errorf("Query failed: %v", err)
				}
				res.Close()
			},
			wantErr: false,
		},
		{
			name: "Query Failure",
			testFn: func(t *testing.T) {
				mdb.Mock.ExpectQuery("SELECT").WillReturnError(errors.New("query error"))

				_, err := wrapper.Query(ctx, "test-op", "SELECT * FROM test")
				if err == nil {
					t.Error("Expected error, got nil")
				}
			},
			wantErr: true,
		},
		{
			name: "QueryRow",
			testFn: func(t *testing.T) {
				mdb.Mock.ExpectQuery("SELECT").WillReturnRows(mdb.NewRows([]string{"id"}).AddRow(1))
				row := wrapper.QueryRow(ctx, "test-op", "SELECT id FROM test")
				var id int
				if err := row.Scan(&id); err != nil {
					t.Errorf("QueryRow Scan failed: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "Array Helper",
			testFn: func(t *testing.T) {
				arr := wrapper.Array([]string{"a", "b"})
				if arr == nil {
					t.Error("Expected non-nil array wrapper")
				}
			},
			wantErr: false,
		},
		{
			name: "AnyArg",
			testFn: func(t *testing.T) {
				if mdb.AnyArg() == nil {
					t.Error("Expected non-nil AnyArg")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFn(t)
		})
	}
}

func TestGetPostgresDSN(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		mockStore *simpleMockStore
		want      string
		wantErr   bool
	}{
		{
			name: "DATABASE_URL Env",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@host/db",
			},
			want:    "postgres://user:pass@host/db",
			wantErr: false,
		},
		{
			name: "Missing Credentials",
			env: map[string]string{
				"DATABASE_URL":       "",
				"SERVER_DB_PASSWORD": "",
			},
			mockStore: &simpleMockStore{values: map[string]string{}},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				if v == "" {
					os.Unsetenv(k)
				} else {
					os.Setenv(k, v)
				}
			}
			defer func() {
				for k := range tt.env {
					os.Unsetenv(k)
				}
			}()

			dsn, _, _, err := GetPostgresDSN(tt.mockStore)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPostgresDSN() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && dsn != tt.want {
				t.Errorf("GetPostgresDSN() got = %v, want %v", dsn, tt.want)
			}
		})
	}

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

	tests := []struct {
		name   string
		testFn func()
	}{
		{
			name: "ExpectTableCreation",
			testFn: func() {
				mdb.ExpectTableCreation("test_table")
			},
		},
		{
			name: "ExpectHypertableCreation",
			testFn: func() {
				mdb.ExpectHypertableCreation("test_table")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFn()
		})
	}
}
