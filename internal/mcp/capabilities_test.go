package mcp

import (
	"context"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCapabilityRegistry_Snapshot(t *testing.T) {
	registry := NewCapabilityRegistry("1.0.0")
	registry.AddAvailable("mcp.network", []string{"observe_network_flows"})
	registry.AddSkipped("mcp.pods", "failed to load kubeconfig")

	snapshot := registry.Snapshot()
	if snapshot.ServerVersion != "1.0.0" {
		t.Fatalf("got version %q, want 1.0.0", snapshot.ServerVersion)
	}
	if snapshot.ToolCount != 1 {
		t.Fatalf("got tool count %d, want 1", snapshot.ToolCount)
	}
	if len(snapshot.Domains) != 2 {
		t.Fatalf("got %d domains, want 2", len(snapshot.Domains))
	}
	if snapshot.Domains[0].Status != "available" {
		t.Fatalf("got status %q, want available", snapshot.Domains[0].Status)
	}
	if snapshot.Domains[1].Reason == "" {
		t.Fatal("expected skipped domain reason")
	}
}

func TestHandleMCPCapabilities(t *testing.T) {
	registry := NewCapabilityRegistry("1.0.0")
	registry.AddAvailable("mcp.diagnostics", []string{"mcp_capabilities"})
	registry.AddSkipped("mcp.telemetry", "missing telemetry environment")

	handler := handleMCPCapabilities(registry, "svc")
	res, _, err := handler(context.Background(), &sdkmcp.CallToolRequest{}, struct{}{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	text := res.Content[0].(*sdkmcp.TextContent).Text
	for _, want := range []string{`"tool_count":1`, `"name":"mcp.telemetry"`, `"status":"skipped"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("got %s, want to contain %s", text, want)
		}
	}
}
