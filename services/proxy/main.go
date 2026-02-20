package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"env"
	"proxy/utils"
	"secrets"
	"telemetry"
)

type App struct {
	server           *http.Server
	SecretProviderFn func() (secrets.SecretStore, error)
}

func main() {
	app := &App{
		SecretProviderFn: func() (secrets.SecretStore, error) {
			return secrets.NewBaoProvider()
		},
	}
	if err := app.Bootstrap(context.Background()); err != nil {
		telemetry.Error("bootstrap_failed", "error", err)
		os.Exit(1)
	}
}

func (a *App) Bootstrap(ctx context.Context) error {
	// 1. Telemetry (gracefully degrades if OTEL_EXPORTER_OTLP_ENDPOINT is not set)
	shutdownTracer, shutdownMeter, shutdownLogger, err := telemetry.Init(ctx, "proxy")
	if err != nil {
		telemetry.Warn("otel_init_failed, continuing without full observability", "error", err)
	}
	defer func() {
		if shutdownTracer != nil {
			if err := shutdownTracer(ctx); err != nil {
				telemetry.Error("otel_shutdown_failed", "component", "tracer", "error", err)
			}
		}
		if shutdownMeter != nil {
			if err := shutdownMeter(ctx); err != nil {
				telemetry.Error("otel_shutdown_failed", "component", "meter", "error", err)
			}
		}
		if shutdownLogger != nil {
			if err := shutdownLogger(ctx); err != nil {
				telemetry.Error("otel_shutdown_failed", "component", "logger", "error", err)
			}
		}
	}()

	env.Load()

	// 2. Secrets
	_, err = a.SecretProviderFn()
	if err != nil {
		return fmt.Errorf("secret_provider_init_failed: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	// 3. Routes with OTel-instrumented mux
	mux := http.NewServeMux()
	mux.HandleFunc("/", utils.WithLogging(utils.HomeHandler))
	mux.HandleFunc("/api/health", utils.WithLogging(utils.HealthHandler))
	mux.HandleFunc("/api/webhook/gitops", utils.WithLogging(utils.WebhookHandler))
	mux.HandleFunc("/api/trace/synthetic/", utils.WithLogging(utils.SyntheticTraceHandler))

	handler := telemetry.NewHTTPHandler(mux, "proxy")

	telemetry.Info("🚀 The GO proxy listening on port", "port", port)

	if os.Getenv("APP_ENV") == "test" {
		return nil
	}

	a.server = &http.Server{Addr: ":" + port, Handler: handler}
	return a.server.ListenAndServe()
}
