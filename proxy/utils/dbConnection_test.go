package utils

import (
	"database/sql"
	"database/sql/driver"
	"os"
	"os/exec"
	"testing"
)

// --- Mock Driver Implementation ---

type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return nil, nil
}

func (c *mockConn) Close() error {
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	return nil, nil
}

// init registers the mock driver so it can be used in tests
func init() {
	sql.Register("mock-postgres", &mockDriver{})
}

// --- Tests ---

func TestInitPostgres_Success(t *testing.T) {
	// Set required environment variables
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_NAME", "testdb")
	// Optional
	os.Setenv("DB_PORT", "5432")
	os.Setenv("SERVER_DB_PASSWORD", "secret")

	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("SERVER_DB_PASSWORD")
	}()

	db := InitPostgres("mock-postgres")

	if db == nil {
		t.Fatal("Expected db instance, got nil")
	}
	defer db.Close()
}

func TestInitPostgres_MissingEnv(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		os.Unsetenv("DB_HOST") // Ensure it's missing
		InitPostgres("mock-postgres")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestInitPostgres_MissingEnv")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestInitMongo_MissingEnv(t *testing.T) {
	if os.Getenv("MONGO_CRASHER") == "1" {
		os.Unsetenv("MONGO_URI")
		InitMongo()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestInitMongo_MissingEnv")
	cmd.Env = append(os.Environ(), "MONGO_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestInitMongo_ConnectionFail(t *testing.T) {
	// Tests that InitMongo fails (log.Fatalf) when it cannot connect/ping.
	// We use a dummy URI that should cause Ping to fail.
	if os.Getenv("MONGO_PING_CRASHER") == "1" {
		os.Setenv("MONGO_URI", "mongodb://localhost:27017") // Assuming no mongo running
		InitMongo()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestInitMongo_ConnectionFail")
	cmd.Env = append(os.Environ(), "MONGO_PING_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
