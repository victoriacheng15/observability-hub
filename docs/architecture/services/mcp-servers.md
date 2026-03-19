# MCP Gateway Architecture

The Observability Hub implements a unified **Model Context Protocol (MCP)** gateway to bridge the gap between AI agents and the platform's specialized domains. This "Agentic Interface" allows LLM-based tools (Gemini CLI, GitHub Copilot) to autonomously interact with system telemetry, Kubernetes infrastructure, and host-level automation through a single authoritative entry point.

## 🎯 Objective

To provide a standardized, intent-based interface for autonomous operations. Instead of requiring human engineers to manually correlate data across multiple UIs, the MCP gateway exposes high-level "Tools" that agents can use to perform multi-domain analysis, trigger reconciliation, and investigate incidents via a unified reasoning loop.

## 🧩 Unified Domain Inventory

The gateway consolidates capabilities into a single binary (`mcp_obs_hub`) while maintaining logical isolation via specialized providers:

| Domain | Provider | Purpose | Key Tools |
| :--- | :--- | :--- | :--- |
| **Telemetry** | `mcp.telemetry` | **Health Brain**: Bridges the LGTM stack for autonomous observability. | `query_metrics`, `query_logs`, `query_traces`, `investigate_incident` |
| **Kubernetes**| `mcp.pods` | **Infrastructure Brain**: Provides high-fidelity cluster state for pod and event analysis. | `inspect_pods`, `describe_pod`, `list_pod_events`, `get_pod_logs`, `delete_pod` |
| **Host/Hub** | `mcp.hub` | **System Brain**: Direct host-level intelligence for systemd and hardware state. | `hub_inspect_platform`, `hub_inspect_host`, `hub_list_host_services`, `hub_query_service_logs` |

## ⚙️ Architectural Standards

The MCP gateway adheres to a consistent, consolidated architectural standard:

- **Protocol**: Model Context Protocol (MCP) over Stdio for seamless integration with local agent runtimes.
- **Fat Binary Architecture**: Multi-domain logic is collapsed into a single high-performance binary to minimize management overhead and streamline the build/deploy pipeline.
- **Soft-Fail Initialization**: Providers initialize sequentially; the gateway remains operational even if specific backends (e.g., a specific database or cluster API) are temporarily unreachable.
- **Unified Instrumentation**: The gateway is instrumented with the platform's Go SDK, emitting logs, metrics, and traces via OTLP to the central OpenTelemetry Collector using the `mcp.service` attribute as a domain discriminator.
- **Decoupled Logic**: Tool handlers are decoupled into `internal/mcp/tools`, while domain access is abstracted into `internal/mcp/providers`, ensuring clean architectural boundaries.

## 🔭 Logic & Data Flow

1. **Initialization**: The gateway initializes the OTel SDK and sequentially registers the Hub, Pods, and Telemetry providers.
2. **Registration**: 13 specialized tools are registered with the MCP SDK, defining strict JSON schemas for intent-based inputs.
3. **Execution**: When an agent invokes a tool, the gateway routes the request to the appropriate provider, captures results, and returns structured content.
4. **Tracing**: Every tool invocation generates a trace span, correlating the agent's intent with the underlying system operations (e.g., `mcp.tool.query_metrics`).

## 🔌 Integration Mapping

| Interface | Protocol | Connectivity | Role |
| :--- | :--- | :--- | :--- |
| **Agent Inbound** | MCP (Stdio) | Local Process | Unified reasoning interface |
| **Telemetry Outbound**| HTTP/gRPC | `localhost:<NodePort>` | Data tier access (Thanos/Loki/Tempo) |
| **Cluster Outbound**| HTTPS | `K3s API` | Infrastructure state access |
| **Host Outbound** | D-Bus/Systemd| Local Socket | System management access |
| **Self-Observability** | OTLP (gRPC) | `localhost:30317` | Telemetry pipeline (OTLP) |
