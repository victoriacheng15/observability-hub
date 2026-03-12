package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	internalmcp "observability-hub/internal/mcp"
	"observability-hub/internal/mcp/providers"
	"observability-hub/internal/telemetry"
)

const (
	serviceName = "mcp-hub"
	version     = "0.1.0"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Initialize Telemetry
	shutdown, err := telemetry.Init(ctx, serviceName)
	if err != nil {
		telemetry.Error("failed to initialize telemetry", "error", err)
		os.Exit(1)
	}
	defer shutdown()

	// 2. Initialize Hub Provider
	hubProvider := providers.NewHubProvider()

	// 3. Create MCP Server using established implementation
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serviceName,
		Version: version,
	}, nil)

	// 4. Register Hub Tools
	internalmcp.RegisterHubTools(server, hubProvider)

	// 5. Run Server (Stdio transport)
	telemetry.Info("mcp-hub ready", "tools", []string{"hub_inspect_platform", "hub_inspect_host", "hub_list_host_services", "hub_query_service_logs"})

	transport := &mcp.StdioTransport{}
	if err := server.Run(ctx, transport); err != nil {
		telemetry.Error("mcp-hub execution failed", "error", err)
		os.Exit(1)
	}

	telemetry.Info("shutting down mcp-hub")
}
