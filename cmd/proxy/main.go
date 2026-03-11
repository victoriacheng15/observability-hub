package main

import (
	"context"
	"os"

	"observability-hub/internal/proxy"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
)

func main() {
	app := proxy.NewApp(func() (secrets.SecretStore, error) {
		return secrets.NewBaoProvider()
	})
	if err := app.Bootstrap(context.Background()); err != nil {
		telemetry.Error("bootstrap_failed", "error", err)
		os.Exit(1)
	}
}
