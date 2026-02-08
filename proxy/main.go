package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"db/mongodb"
	"db/postgres"
	"logger"
	"proxy/utils"
	"secrets"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type App struct {
	server           *http.Server
	SecretProviderFn func() (secrets.SecretStore, error)
	PostgresConnFn   func(driver string, store secrets.SecretStore) (*postgres.ReadingStore, error)
	MongoConnFn      func(store secrets.SecretStore) (*mongodb.MongoStore, error)
}

func main() {
	app := &App{
		SecretProviderFn: func() (secrets.SecretStore, error) {
			return secrets.NewBaoProvider()
		},
		PostgresConnFn: func(driver string, store secrets.SecretStore) (*postgres.ReadingStore, error) {
			conn, err := postgres.ConnectPostgres(driver, store)
			if err != nil {
				return nil, err
			}
			return postgres.NewReadingStore(conn), nil
		},
		MongoConnFn: func(store secrets.SecretStore) (*mongodb.MongoStore, error) {
			return mongodb.NewMongoStore(store)
		},
	}
	if err := app.Bootstrap(context.Background()); err != nil {
		slog.Error("bootstrap_failed", "error", err)
		os.Exit(1)
	}
}

func (a *App) Bootstrap(ctx context.Context) error {
	logger.Setup(os.Stdout, "proxy")

	godotenv.Load(".env")
	godotenv.Load("../.env")

	// 1. Secrets
	secretStore, err := a.SecretProviderFn()
	if err != nil {
		return fmt.Errorf("secret_provider_init_failed: %w", err)
	}

	// 2. Postgres
	readingStore, err := a.PostgresConnFn("postgres", secretStore)
	if err != nil {
		return fmt.Errorf("db_connection_failed (postgres): %w", err)
	}
	slog.Info("db_connected", "database", "postgres")

	// 3. Mongo
	mongoStore, err := a.MongoConnFn(secretStore)
	if err != nil {
		return fmt.Errorf("db_connection_failed (mongodb): %w", err)
	}
	slog.Info("db_connected", "database", "mongodb")

	// 4. Reading Service
	readingService := &utils.ReadingService{
		Store:      readingStore,
		MongoStore: mongoStore,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	http.HandleFunc("/", utils.WithLogging(utils.HomeHandler))
	http.HandleFunc("/api/health", utils.WithLogging(utils.HealthHandler))
	http.HandleFunc("/api/sync/reading", utils.WithLogging(readingService.SyncReadingHandler))
	http.HandleFunc("/api/webhook/gitops", utils.WithLogging(utils.WebhookHandler))

	slog.Info("🚀 The GO proxy listening on port", "port", port)

	if os.Getenv("APP_ENV") == "test" {
		return nil
	}

	a.server = &http.Server{Addr: ":" + port}
	return a.server.ListenAndServe()
}
