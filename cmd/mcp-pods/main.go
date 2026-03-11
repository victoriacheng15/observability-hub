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

const serviceName = "mcp.pods"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env.Load()

	if shutdown, err := telemetry.Init(ctx, serviceName); err != nil {
		telemetry.Error("failed to init telemetry", "error", err)
	} else {
		defer shutdown()
	}

	provider, err := providers.NewPodsProvider()
	if err != nil {
		telemetry.Error("failed to create pods provider", "error", err)
		os.Exit(1)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-pods",
		Version: "1.0.0",
	}, nil)

	internalmcp.RegisterPodsTools(server, provider)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		t := &mcp.StdioTransport{}
		if err := server.Run(ctx, t); err != nil {
			telemetry.Error("MCP server stopped", "error", err)
		}
	}()

	telemetry.Info("mcp-pods ready", "tools", []string{"inspect_pods", "describe_pod", "list_pod_events", "get_pod_logs", "delete_pod"})

	sig := <-sigChan
	telemetry.Info("received signal, shutting down", "signal", sig.String())
	cancel()
}
