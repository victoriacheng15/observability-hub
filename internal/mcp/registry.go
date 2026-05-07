package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-hub/internal/mcp/providers"
	"observability-hub/internal/mcp/tools/network"
	"observability-hub/internal/mcp/tools/pods"
	"observability-hub/internal/mcp/tools/telemetry"
	libtelemetry "observability-hub/internal/telemetry"
)

// --- Telemetry Tools ---

// RegisterTelemetryTools registers all telemetry-related tools (Prometheus, Loki, Tempo) to the MCP server.
func RegisterTelemetryTools(server *mcp.Server, provider *providers.TelemetryProvider, serviceName string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_metrics",
		Description: "Execute PromQL queries against Prometheus for metrics analysis (See skills/telemetry/SKILL.md for guidance)",
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

	libtelemetry.Info("registered telemetry tools", "count", 4)
}

func handleQueryMetrics(provider *providers.TelemetryProvider, serviceName string) mcp.ToolHandlerFor[telemetry.QueryMetricsInput, any] {
	handler := telemetry.NewQueryMetricsHandler(provider.QueryMetrics)
	return NewJSONToolHandler("query_metrics", serviceName, handler.Execute)
}

func handleQueryLogs(provider *providers.TelemetryProvider, serviceName string) mcp.ToolHandlerFor[telemetry.QueryLogsInput, any] {
	handler := telemetry.NewQueryLogsHandler(provider.QueryLogs)
	return NewJSONToolHandler("query_logs", serviceName, handler.Execute)
}

func handleInvestigateIncident(provider *providers.TelemetryProvider, serviceName string) mcp.ToolHandlerFor[telemetry.InvestigateIncidentInput, any] {
	handler := telemetry.NewInvestigateIncidentHandler(provider.QueryMetrics, provider.QueryLogs, provider.QueryTraces)
	return NewJSONToolHandler("investigate_incident", serviceName, handler.Execute)
}

func handleQueryTraces(provider *providers.TelemetryProvider, serviceName string) mcp.ToolHandlerFor[telemetry.QueryTracesInput, any] {
	handler := telemetry.NewQueryTracesHandler(provider.QueryTraces)
	return NewJSONToolHandler("query_traces", serviceName, handler.Execute)
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

	libtelemetry.Info("registered pods tools", "count", 5)
}

func handleInspectPods(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[pods.PodsInput, any] {
	handler := pods.NewInspectPodsHandler(provider.ListPods)
	return NewJSONToolHandler("inspect_pods", serviceName, handler.Execute)
}

func handleDescribePod(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[pods.PodsInput, any] {
	handler := pods.NewDescribePodHandler(provider.GetPod)
	return NewJSONToolHandler("describe_pod", serviceName, handler.Execute)
}

func handleListPodEvents(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[pods.PodsInput, any] {
	handler := pods.NewListPodEventsHandler(provider.ListEvents)
	return NewJSONToolHandler("list_pod_events", serviceName, handler.Execute)
}

func handleGetPodLogs(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[pods.PodLogsInput, any] {
	handler := pods.NewGetPodLogsHandler(provider.GetPodLogs)
	return NewTextToolHandler("get_pod_logs", serviceName, func(ctx context.Context, input pods.PodLogsInput) (string, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return "", err
		}
		text, ok := result.(string)
		if !ok {
			return "", fmt.Errorf("get_pod_logs returned %T, want string", result)
		}
		return text, nil
	})
}

func handleDeletePod(provider *providers.PodsProvider, serviceName string) mcp.ToolHandlerFor[pods.DeletePodInput, any] {
	handler := pods.NewDeletePodHandler(provider.DeletePod)
	return NewJSONToolHandler("delete_pod", serviceName, handler.Execute)
}

// --- Network Tools ---

// RegisterNetworkTools registers all networking-related tools (Hubble) to the MCP server.
func RegisterNetworkTools(server *mcp.Server, provider *providers.NetworkProvider, serviceName string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "observe_network_flows",
		Description: "Query real-time network flows from Hubble Relay (See skills/network/SKILL.md for guidance)",
	}, handleObserveNetworkFlows(provider, serviceName))

	libtelemetry.Info("registered network tools", "count", 1)
}

func handleObserveNetworkFlows(provider *providers.NetworkProvider, serviceName string) mcp.ToolHandlerFor[network.ObserveNetworkFlowsInput, any] {
	handler := network.NewObserveNetworkFlowsHandler(provider.QueryHubbleFlows)
	return NewTextToolHandler("observe_network_flows", serviceName, func(ctx context.Context, input network.ObserveNetworkFlowsInput) (string, error) {
		result, err := handler.Execute(ctx, input)
		if err != nil {
			return "", err
		}
		text, ok := result.(string)
		if !ok {
			return "", fmt.Errorf("observe_network_flows returned %T, want string", result)
		}
		return text, nil
	})
}
