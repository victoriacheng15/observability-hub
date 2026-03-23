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
func RegisterTelemetryTools(server *mcp.Server, provider *providers.TelemetryProvider, serviceName string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_metrics",
		Description: "Execute PromQL queries against Thanos/Prometheus for metrics analysis (See skills/telemetry/SKILL.md for guidance)",
	}, handleQueryMetrics(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_logs",
		Description: "Execute LogQL queries against Loki for log analysis (See skills/telemetry/SKILL.md for guidance)",
	}, handleQueryLogs(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_traces",
		Description: "Retrieve distributed traces from Tempo by trace ID (See skills/telemetry/SKILL.md for guidance)",
	}, handleQueryTraces(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "investigate_incident",
		Description: "Correlate metrics, logs, and traces to produce a structured incident report for a service (See skills/telemetry/SKILL.md for guidance)",
	}, handleInvestigateIncident(provider, serviceName))

	telemetry.Info("registered telemetry tools", "count", 4)
}

func handleQueryMetrics(provider *providers.TelemetryProvider, serviceName string) mcp.ToolHandlerFor[tools.QueryMetricsInput, any] {
	handler := tools.NewQueryMetricsHandler(provider.QueryMetrics)
	return InstrumentHandler("query_metrics", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.QueryMetricsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleQueryLogs(provider *providers.TelemetryProvider, serviceName string) mcp.ToolHandlerFor[tools.QueryLogsInput, any] {
	handler := tools.NewQueryLogsHandler(provider.QueryLogs)
	return InstrumentHandler("query_logs", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.QueryLogsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleInvestigateIncident(provider *providers.TelemetryProvider, serviceName string) mcp.ToolHandlerFor[tools.InvestigateIncidentInput, any] {
	handler := tools.NewInvestigateIncidentHandler(provider.QueryMetrics, provider.QueryLogs, provider.QueryTraces)
	return InstrumentHandler("investigate_incident", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.InvestigateIncidentInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleQueryTraces(provider *providers.TelemetryProvider, serviceName string) mcp.ToolHandlerFor[tools.QueryTracesInput, any] {
	handler := tools.NewQueryTracesHandler(provider.QueryTraces)
	return InstrumentHandler("query_traces", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.QueryTracesInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

// --- Pods Tools ---

// RegisterPodsTools registers all Kubernetes-related tools (Pods, Events) to the MCP server.
func RegisterPodsTools(server *mcp.Server, provider *providers.PodsProvider, serviceName string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "inspect_pods",
		Description: "List all pods in a namespace with status summary (See skills/pods/SKILL.md for guidance)",
	}, handleInspectPods(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_pod",
		Description: "Get detailed status and configuration for a specific pod (See skills/pods/SKILL.md for guidance)",
	}, handleDescribePod(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_pod_events",
		Description: "List all lifecycle events associated with a specific pod (See skills/pods/SKILL.md for guidance)",
	}, handleListPodEvents(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_pod_logs",
		Description: "Retrieve logs from a specific pod/container (See skills/pods/SKILL.md for guidance)",
	}, handleGetPodLogs(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_pod",
		Description: "Delete a specific pod (useful for restarting stuck pods) (See skills/pods/SKILL.md for guidance)",
	}, handleDeletePod(provider, serviceName))

	telemetry.Info("registered pods tools", "count", 5)
}

func handleInspectPods(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[tools.PodsInput, any] {
	handler := tools.NewInspectPodsHandler(provider.ListPods)
	return InstrumentHandler("inspect_pods", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.PodsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleDescribePod(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[tools.PodsInput, any] {
	handler := tools.NewDescribePodHandler(provider.GetPod)
	return InstrumentHandler("describe_pod", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.PodsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleListPodEvents(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[tools.PodsInput, any] {
	handler := tools.NewListPodEventsHandler(provider.ListEvents)
	return InstrumentHandler("list_pod_events", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.PodsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleGetPodLogs(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[tools.PodLogsInput, any] {
	handler := tools.NewGetPodLogsHandler(provider.GetPodLogs)
	return InstrumentHandler("get_pod_logs", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.PodLogsInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result.(string)}},
		}, nil, nil
	})
}

func handleDeletePod(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[tools.DeletePodInput, any] {
	handler := tools.NewDeletePodHandler(provider.DeletePod)
	return InstrumentHandler("delete_pod", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.DeletePodInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

// --- Hub Tools ---

// RegisterHubTools registers all host-level and platform status tools to the MCP server.
func RegisterHubTools(server *mcp.Server, provider *providers.HubProvider, serviceName string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hub_inspect_platform",
		Description: "Get an executive summary of the entire platform health (See skills/platform/SKILL.md for guidance)",
	}, handleInspectPlatform(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hub_inspect_host",
		Description: "Inspect physical resource pressure (Load, Memory, Disk) on the main server (See skills/host/SKILL.md for guidance)",
	}, handleInspectHost(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hub_list_host_services",
		Description: "List and check status of core systemd units (ingestion, proxy, openbao) (See skills/host/SKILL.md for guidance)",
	}, handleListHostServices(provider, serviceName))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hub_query_service_logs",
		Description: "Query systemd journal logs for a specific service since a relative time (e.g., past 5m, 1h) (See skills/host/SKILL.md for guidance)",
	}, handleQueryServiceLogs(provider, serviceName))

	telemetry.Info("registered hub tools", "count", 4)
}

func handleInspectPlatform(provider *providers.HubProvider, serviceName string) mcp.ToolHandlerFor[tools.HubInput, any] {
	handler := tools.NewInspectPlatformHandler(provider.InspectPlatform)
	return InstrumentHandler("hub_inspect_platform", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.HubInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleInspectHost(provider *providers.HubProvider, serviceName string) mcp.ToolHandlerFor[tools.HubInput, any] {
	handler := tools.NewInspectHostHandler(provider.InspectHost)
	return InstrumentHandler("hub_inspect_host", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.HubInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleListHostServices(provider *providers.HubProvider, serviceName string) mcp.ToolHandlerFor[tools.HubInput, any] {
	handler := tools.NewListHostServicesHandler(provider.ListHostServices)
	return InstrumentHandler("hub_list_host_services", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.HubInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		}, nil, nil
	})
}

func handleQueryServiceLogs(provider *providers.HubProvider, serviceName string) mcp.ToolHandlerFor[tools.HubInput, any] {
	handler := tools.NewQueryServiceLogsHandler(provider.QueryServiceLogs)
	return InstrumentHandler("hub_query_service_logs", serviceName, func(ctx context.Context, _ *mcp.CallToolRequest, input tools.HubInput) (*mcp.CallToolResult, any, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result.(string)}},
		}, nil, nil
	})
}
