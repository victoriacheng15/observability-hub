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

var (
	networkTools    = []string{"observe_network_flows"}
	podsTools       = []string{"inspect_pods", "describe_pod", "list_pod_events", "get_pod_logs", "delete_pod"}
	telemetryTools  = []string{"query_metrics", "query_logs", "query_traces", "investigate_incident"}
	diagnosticTools = []string{"mcp_capabilities"}
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
	capabilities := internalmcp.NewCapabilityRegistry(version)

	// 3. Sequential Provider Initialization (Soft-Fail Pattern)

	// --- Network Provider ---
	networkProv := providers.NewNetworkProvider()
	if networkProv != nil {
		internalmcp.RegisterNetworkTools(server, networkProv, "mcp.network")
		capabilities.AddAvailable("mcp.network", networkTools)
		telemetry.Info("registered network tools", "count", len(networkTools))
	} else {
		reason := "network provider initialization returned nil"
		capabilities.AddSkipped("mcp.network", reason)
		telemetry.Warn("mcp_network_init_failed_skipping_tools", "reason", reason)
	}

	// --- Pods Provider ---
	if podsProv, err := providers.NewPodsProvider(); err != nil {
		capabilities.AddSkipped("mcp.pods", err.Error())
		telemetry.Warn("mcp_pods_init_failed_skipping_tools", "error", err)
	} else {
		internalmcp.RegisterPodsTools(server, podsProv, "mcp.pods")
		capabilities.AddAvailable("mcp.pods", podsTools)
		telemetry.Info("registered pods tools (mcp.pods)", "count", len(podsTools))
	}

	// --- Telemetry Provider ---
	thanosURL := os.Getenv("THANOS_URL")
	lokiURL := os.Getenv("LOKI_URL")
	tempoURL := os.Getenv("TEMPO_URL")

	if thanosURL == "" || lokiURL == "" || tempoURL == "" {
		reason := "THANOS_URL, LOKI_URL, or TEMPO_URL is missing"
		capabilities.AddSkipped("mcp.telemetry", reason)
		telemetry.Warn("mcp_telemetry_init_failed_missing_env_skipping_tools", "reason", reason)
	} else {
		telemetryProv := providers.NewTelemetryProvider(thanosURL, lokiURL, tempoURL)
		if telemetryProv != nil {
			defer telemetryProv.Close()
			internalmcp.RegisterTelemetryTools(server, telemetryProv, "mcp.telemetry")
			capabilities.AddAvailable("mcp.telemetry", telemetryTools)
			telemetry.Info("registered telemetry tools (mcp.telemetry)", "count", len(telemetryTools))
		} else {
			reason := "telemetry provider initialization returned nil"
			capabilities.AddSkipped("mcp.telemetry", reason)
			telemetry.Warn("mcp_telemetry_init_failed_skipping_tools", "reason", reason)
		}
	}

	capabilities.AddAvailable("mcp.diagnostics", diagnosticTools)
	internalmcp.RegisterCapabilityTools(server, capabilities, "mcp.diagnostics")

	// 4. Run Server (Stdio transport)
	snapshot := capabilities.Snapshot()
	telemetry.Info(
		"mcp-obs-hub ready",
		"tool_count", snapshot.ToolCount,
		"domain_count", len(snapshot.Domains),
		"domains", snapshot.Domains,
	)

	transport := &mcp.StdioTransport{}
	if err := server.Run(ctx, transport); err != nil {
		telemetry.Error("mcp-obs-hub execution failed", "error", err)
		os.Exit(1)
	}

	telemetry.Info("shutting down mcp-obs-hub")
}
