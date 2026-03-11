package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-hub/internal/mcp/providers"
	"observability-hub/internal/mcp/tools"
	"observability-hub/internal/telemetry"
)

// --- Telemetry Tools ---

// RegisterTelemetryTools registers all telemetry-related tools (Thanos, Loki, Tempo) to the MCP server.
func RegisterTelemetryTools(server *mcp.Server, provider *providers.TelemetryProvider) {
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

	telemetry.Info("registered telemetry tools", "count", 4)
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

// --- Pods Tools ---

// RegisterPodsTools registers all Kubernetes-related tools (Pods, Events) to the MCP server.
func RegisterPodsTools(server *mcp.Server, provider *providers.PodsProvider) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "inspect_pods",
		Description: "List all pods in a namespace with status summary",
	}, handleInspectPods(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_pod",
		Description: "Get detailed status and configuration for a specific pod",
	}, handleDescribePod(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_pod_events",
		Description: "List all lifecycle events associated with a specific pod",
	}, handleListPodEvents(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_pod_logs",
		Description: "Retrieve logs from a specific pod/container",
	}, handleGetPodLogs(provider))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_pod",
		Description: "Delete a specific pod (useful for restarting stuck pods)",
	}, handleDeletePod(provider))

	telemetry.Info("registered pods tools", "count", 5)
}

func handleInspectPods(provider *providers.PodsProvider) mcp.ToolHandlerFor[tools.PodsInput, any] {
	handler := tools.NewInspectPodsHandler(provider.ListPods)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.PodsInput) (*mcp.CallToolResult, any, error) {
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

func handleDescribePod(provider *providers.PodsProvider) mcp.ToolHandlerFor[tools.PodsInput, any] {
	handler := tools.NewDescribePodHandler(provider.GetPod)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.PodsInput) (*mcp.CallToolResult, any, error) {
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

func handleListPodEvents(provider *providers.PodsProvider) mcp.ToolHandlerFor[tools.PodsInput, any] {
	handler := tools.NewListPodEventsHandler(provider.ListEvents)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.PodsInput) (*mcp.CallToolResult, any, error) {
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

func handleGetPodLogs(provider *providers.PodsProvider) mcp.ToolHandlerFor[tools.PodLogsInput, any] {
	handler := tools.NewGetPodLogsHandler(provider.GetPodLogs)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.PodLogsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result.(string)}},
		}, nil, nil
	}
}

func handleDeletePod(provider *providers.PodsProvider) mcp.ToolHandlerFor[tools.DeletePodInput, any] {
	handler := tools.NewDeletePodHandler(provider.DeletePod)
	return func(ctx context.Context, _ *mcp.CallToolRequest, input tools.DeletePodInput) (*mcp.CallToolResult, any, error) {
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
