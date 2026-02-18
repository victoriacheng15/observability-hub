package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"db/mongodb"
	"db/postgres"
	"env"
	"logger"
	"proxy/utils"
	"secrets"
	"telemetry"
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

	env.Load()

	// 1. Telemetry (gracefully degrades if OTEL_EXPORTER_OTLP_ENDPOINT is not set)
	shutdownTracer, shutdownMeter, shutdownLogger, err := telemetry.Init(ctx)
	if err != nil {
		slog.Warn("otel_init_failed, continuing without full observability", "error", err)
	}
	defer func() {
		if shutdownTracer != nil {
			if err := shutdownTracer(ctx); err != nil {
				slog.Error("otel_shutdown_failed", "component", "tracer", "error", err)
			}
		}
		if shutdownMeter != nil {
			if err := shutdownMeter(ctx); err != nil {
				slog.Error("otel_shutdown_failed", "component", "meter", "error", err)
			}
		}
		if shutdownLogger != nil {
			if err := shutdownLogger(ctx); err != nil {
				slog.Error("otel_shutdown_failed", "component", "logger", "error", err)
			}
		}
	}()

	// 2. Secrets
	secretStore, err := a.SecretProviderFn()
	if err != nil {
		return fmt.Errorf("secret_provider_init_failed: %w", err)
	}

	// 3. Postgres
	readingStore, err := a.PostgresConnFn("postgres", secretStore)
	if err != nil {
		return fmt.Errorf("db_connection_failed (postgres): %w", err)
	}
	slog.Info("db_connected", "database", "postgres")

	// 4. Mongo
	mongoStore, err := a.MongoConnFn(secretStore)
	if err != nil {
		return fmt.Errorf("db_connection_failed (mongodb): %w", err)
	}
	slog.Info("db_connected", "database", "mongodb")

	// 5. Reading Service
	readingService := &utils.ReadingService{
		Store:      readingStore,
		MongoStore: mongoStore,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	// 6. Routes with OTel-instrumented mux
	mux := http.NewServeMux()
	mux.HandleFunc("/", utils.WithLogging(utils.HomeHandler))
	mux.HandleFunc("/api/health", utils.WithLogging(utils.HealthHandler))
	mux.HandleFunc("/api/sync/reading", utils.WithLogging(readingService.SyncReadingHandler))
	mux.HandleFunc("/api/webhook/gitops", utils.WithLogging(utils.WebhookHandler))
	mux.HandleFunc("/api/trace/synthetic/", utils.WithLogging(utils.SyntheticTraceHandler))

	handler := telemetry.NewHTTPHandler(mux, "proxy")

	slog.Info("🚀 The GO proxy listening on port", "port", port)

	if os.Getenv("APP_ENV") == "test" {
		return nil
	}

	a.server = &http.Server{Addr: ":" + port, Handler: handler}
	return a.server.ListenAndServe()
}
