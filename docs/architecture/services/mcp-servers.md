# MCP Servers Architecture

The Observability Hub implements a suite of **Model Context Protocol (MCP)** servers to bridge the gap between AI agents and the platform's specialized domains. These servers act as "Agentic Interfaces," allowing LLM-based tools (GitHub Copilot, Gemini CLI) to autonomously interact with system data and automation.

## 🎯 Objective

To provide a standardized, intent-based interface for autonomous operations. Instead of requiring human engineers to manually correlate data across multiple UIs, MCP servers expose high-level "Tools" that agents can use to perform analysis, trigger reconciliation, and investigate incidents.

## 🧩 Component Inventory

| Service Name | Path | Purpose | Key Tools |
| :--- | :--- | :--- | :--- |
| **`mcp-telemetry`** | `cmd/mcp-telemetry/` | **Health Brain**: Bridges the LGTM stack for autonomous observability. | `query_metrics`, `query_logs`, `query_traces` |

## ⚙️ Shared Architectural Patterns

All MCP servers in the platform adhere to a consistent architectural standard:

- **Protocol**: Model Context Protocol (MCP) over Stdio for seamless integration with local agent runtimes.
- **Unified Telemetry**: Each server is instrumented with the platform's Go SDK, emitting logs, metrics, and traces via OTLP to the central OpenTelemetry Collector.
- **Backend Providers**: Logic is decoupled into `internal/mcp/providers`, allowing the same provider logic (e.g., Thanos querying) to be shared across multiple interfaces.
- **Process Management**: Deployed as native Systemd units on the host for high reliability and direct filesystem/NodePort access.

## 🔭 Logic & Data Flow

1. **Initialization**: The server initializes the OTel SDK and establishes connections to the required data tier (e.g., Loki, Thanos via NodePort).
2. **Registration**: Tools are registered with the MCP SDK, defining clear JSON schemas for inputs (e.g., PromQL strings or Trace IDs).
3. **Execution**: When an agent invokes a tool, the server executes the corresponding provider logic, captures the results, and returns them as structured text or JSON content.
4. **Tracing**: Every tool invocation generates a trace span, correlating the agent's intent with the underlying system operations.

## 🔌 Integration Mapping

| Interface | Protocol | Connectivity | Role |
| :--- | :--- | :--- | :--- |
| **Agent Inbound** | MCP (Stdio) | Local Process | Reasoning loop interface |
| **Backend Outbound**| HTTP/gRPC | `localhost:<NodePort>` | Data tier access |
| **Telemetry** | OTLP (gRPC) | `localhost:30317` | Self-observability pipeline |
