package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-hub/internal/mcp/providers"
	"observability-hub/internal/mcp/tools"
	"observability-hub/internal/telemetry"
)

// RegisterTools registers all telemetry tools to the MCP server.
func RegisterTools(server *mcp.Server, provider *providers.TelemetryProvider) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_metrics",
		Description: "Execute PromQL queries against Thanos/Prometheus for metrics analysis",
	}, handleQueryMetrics(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_logs",
		Description: "Execute LogQL queries against Loki for log analysis",
	}, handleQueryLogs(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_traces",
		Description: "Retrieve distributed traces from Tempo by trace ID",
	}, handleQueryTraces(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "investigate_incident",
		Description: "Correlate metrics, logs, and traces to produce a structured incident report for a service",
	}, handleInvestigateIncident(provider))

	telemetry.Info("registered tools", "count", 4)
}

func handleQueryMetrics(provider *providers.TelemetryProvider) mcp.ToolHandlerFor[tools.QueryMetricsInput, any] {
	handler := tools.NewQueryMetricsHandler(provider.QueryMetrics)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.QueryMetricsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	}
}

func handleQueryLogs(provider *providers.TelemetryProvider) mcp.ToolHandlerFor[tools.QueryLogsInput, any] {
	handler := tools.NewQueryLogsHandler(provider.QueryLogs)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.QueryLogsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	}
}

func handleInvestigateIncident(provider *providers.TelemetryProvider) mcp.ToolHandlerFor[tools.InvestigateIncidentInput, any] {
	handler := tools.NewInvestigateIncidentHandler(provider.QueryMetrics, provider.QueryLogs, provider.QueryTraces)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.InvestigateIncidentInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	}
}

func handleQueryTraces(provider *providers.TelemetryProvider) mcp.ToolHandlerFor[tools.QueryTracesInput, any] {
	handler := tools.NewQueryTracesHandler(provider.QueryTraces)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.QueryTracesInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	}
}
