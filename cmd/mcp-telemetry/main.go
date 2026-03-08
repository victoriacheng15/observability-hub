package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-hub/internal/env"
	"observability-hub/internal/mcp/providers"
	"observability-hub/internal/telemetry"
)

const serviceName = "mcp.telemetry"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env.Load()

	if shutdown, err := telemetry.Init(ctx, serviceName); err != nil {
		telemetry.Error("failed to init telemetry", "error", err)
	} else {
		defer shutdown()
	}

	thanosURL := os.Getenv("THANOS_URL")
	lokiURL := os.Getenv("LOKI_URL")
	tempoURL := os.Getenv("TEMPO_URL")

	if thanosURL == "" {
		telemetry.Error("THANOS_URL not set")
		os.Exit(1)
	}
	if lokiURL == "" {
		telemetry.Error("LOKI_URL not set")
		os.Exit(1)
	}
	if tempoURL == "" {
		telemetry.Error("TEMPO_URL not set")
		os.Exit(1)
	}

	provider := providers.NewTelemetryProvider(thanosURL, lokiURL, tempoURL)
	telemetry.Info("MCP Telemetry server initialized", "thanos_url", thanosURL, "loki_url", lokiURL, "tempo_url", tempoURL)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-telemetry",
		Version: "1.0.0",
	}, nil)

	registerTools(server, provider)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		t := &mcp.StdioTransport{}
		if err := server.Run(ctx, t); err != nil {
			telemetry.Error("MCP server stopped", "error", err)
		}
	}()

	telemetry.Info("mcp-telemetry ready", "tools", []string{"query_metrics", "query_logs", "query_traces"})

	sig := <-sigChan
	telemetry.Info("received signal, shutting down", "signal", sig.String())
	cancel()
	provider.Close()
}
