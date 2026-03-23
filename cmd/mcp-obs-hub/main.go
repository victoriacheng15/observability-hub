package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-hub/internal/env"
	internalmcp "observability-hub/internal/mcp"
	"observability-hub/internal/mcp/providers"
	"observability-hub/internal/telemetry"
)

const (
	serviceName = "mcp"
	version     = "1.0.0"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	env.Load()

	// 1. Initialize Global Telemetry
	shutdown, err := telemetry.Init(ctx, serviceName)
	if err != nil {
		telemetry.Error("failed to initialize global telemetry", "error", err)
		os.Exit(1)
	}
	defer shutdown()

	// 2. Create Unified MCP Server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-obs-hub",
		Version: version,
	}, nil)

	// 3. Sequential Provider Initialization (Soft-Fail Pattern)

	// --- Hub Provider ---
	hubProv := providers.NewHubProvider()
	if hubProv != nil {
		internalmcp.RegisterHubTools(server, hubProv, "mcp.hub")
		internalmcp.RegisterNetworkTools(server, hubProv, "mcp.network")
		telemetry.Info("registered hub and network tools")
	} else {
		telemetry.Warn("mcp_hub_init_failed_skipping_tools")
	}

	// --- Pods Provider ---
	if podsProv, err := providers.NewPodsProvider(); err != nil {
		telemetry.Warn("mcp_pods_init_failed_skipping_tools", "error", err)
	} else {
		internalmcp.RegisterPodsTools(server, podsProv, "mcp.pods")
		telemetry.Info("registered pods tools (mcp.pods)")
	}

	// --- Telemetry Provider ---
	thanosURL := os.Getenv("THANOS_URL")
	lokiURL := os.Getenv("LOKI_URL")
	tempoURL := os.Getenv("TEMPO_URL")

	if thanosURL == "" || lokiURL == "" || tempoURL == "" {
		telemetry.Warn("mcp_telemetry_init_failed_missing_env_skipping_tools")
	} else {
		telemetryProv := providers.NewTelemetryProvider(thanosURL, lokiURL, tempoURL)
		if telemetryProv != nil {
			defer telemetryProv.Close()
			internalmcp.RegisterTelemetryTools(server, telemetryProv, "mcp.telemetry")
			telemetry.Info("registered telemetry tools (mcp.telemetry)")
		}
	}

	// 4. Run Server (Stdio transport)
	telemetry.Info("mcp-obs-hub ready, unified 14 tools available")

	transport := &mcp.StdioTransport{}
	if err := server.Run(ctx, transport); err != nil {
		telemetry.Error("mcp-obs-hub execution failed", "error", err)
		os.Exit(1)
	}

	telemetry.Info("shutting down mcp-obs-hub")
}
