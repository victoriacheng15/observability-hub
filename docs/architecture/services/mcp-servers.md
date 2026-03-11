# MCP Servers Architecture

The Observability Hub implements a suite of **Model Context Protocol (MCP)** servers to bridge the gap between AI agents and the platform's specialized domains. These servers act as "Agentic Interfaces," allowing LLM-based tools (GitHub Copilot, Gemini CLI) to autonomously interact with system data and automation.

## 🎯 Objective

To provide a standardized, intent-based interface for autonomous operations. Instead of requiring human engineers to manually correlate data across multiple UIs, MCP servers expose high-level "Tools" that agents can use to perform analysis, trigger reconciliation, and investigate incidents.

## 🧩 Component Inventory

| Service Name | Path | Purpose | Key Tools |
| :--- | :--- | :--- | :--- |
| **`mcp-telemetry`** | `cmd/mcp-telemetry/` | **Health Brain**: Bridges the LGTM stack for autonomous observability. | `query_metrics`, `query_logs`, `query_traces`, `investigate_incident` |
| **`mcp-pods`** | `cmd/mcp-pods/` | **Infrastructure Brain**: Provides high-fidelity cluster state for pod and event analysis. | `inspect_pods`, `describe_pod`, `list_pod_events` |

## ⚙️ Shared Architectural Patterns

All MCP servers in the platform adhere to a consistent, domain-isolated architectural standard (ADR 018):

- **Protocol**: Model Context Protocol (MCP) over Stdio for seamless integration with local agent runtimes.
- **Domain Isolation**: Capabilities are split into specialized, standalone binaries to enforce the Principle of Least Privilege and reduce the security blast radius.
- **Unified Telemetry**: Each server is instrumented with the platform's Go SDK, emitting logs, metrics, and traces via OTLP to the central OpenTelemetry Collector.
- **Backend Providers**: Logic is decoupled into `internal/mcp/providers`, allowing the same provider logic (e.g., Kubernetes API access) to be shared across multiple interfaces while maintaining strict import boundaries.
- **Deployment Pattern**: Deployed as pure binaries communicating over `stdio`, avoiding host-tier service managers like systemd where possible to improve portability and alignment with future containerized orchestration.

## 🔭 Logic & Data Flow

1. **Initialization**: The server initializes the OTel SDK and establishes connections to its specific domain (e.g., Loki/Thanos via NodePort for telemetry, or the K3s API for pods).
2. **Registration**: Tools are registered with the MCP SDK, defining clear JSON schemas for inputs (e.g., PromQL strings or Namespace/Pod names).
3. **Execution**: When an agent invokes a tool, the server executes the corresponding provider logic, captures the results, and returns them as structured text or JSON content.
4. **Tracing**: Every tool invocation generates a trace span, correlating the agent's intent with the underlying system operations.

## 🔌 Integration Mapping

| Interface | Protocol | Connectivity | Role |
| :--- | :--- | :--- | :--- |
| **Agent Inbound** | MCP (Stdio) | Local Process | Reasoning loop interface |
| **Telemetry Outbound**| HTTP/gRPC | `localhost:<NodePort>` | Data tier access |
| **Cluster Outbound**| HTTPS | `K3s API` | Infrastructure state access |
| **Self-Observability** | OTLP (gRPC) | `localhost:30317` | Telemetry pipeline |
