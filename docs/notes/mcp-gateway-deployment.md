# MCP Gateway: Unified Agentic Execution Guide

The `mcp_obs_hub` gateway provides a unified **Model Context Protocol (MCP)** interface for autonomous platform operations. It consolidates the Telemetry (LGTM), Kubernetes (Pods), and Host (Hub) domains into a single authoritative binary, enabling AI agents to correlate system-wide state through a single reasoning loop.

---

## Quick Commands

### Build (Development)

```bash
make mcp-build
```

This command:

1. Compiles `cmd/mcp-obs-hub` → `bin/mcp_obs_hub`

### Run Manually

```bash
./bin/mcp_obs_hub
```

Operational logs are emitted to **stderr** to ensure `stdout` remains dedicated to the JSON-RPC protocol required by MCP.

---

## How It Works

### Unified Architecture

```text
AI Agent (Gemini CLI / Copilot / obs)
└── Spawn Gateway (bin/mcp_obs_hub)
    └── Sequential Provider Initialization:
        ├── Telemetry (Thanos, Loki, Tempo)
        ├── Kubernetes (K3s API)
        └── Host (Systemd / D-Bus)
```

### Service Lifecycle

1. **Initialization**:
   - Reads `.env` (loads `THANOS_URL`, `LOKI_URL`, `TEMPO_URL`).
   - Initializes OTel telemetry (traces, metrics, logs → OTLP gRPC @ localhost:30317).
   - **Soft-Fail Registration**: Sequential initialization of Hub, Pods, and Telemetry providers. The gateway remains operational even if a specific backend is unreachable.

2. **Execution**:
   - Listens for MCP tool calls over `stdin`.
   - Routes requests to the appropriate domain provider.
   - Returns structured results over `stdout`.
   - **Instrumentation**: Every tool invocation is traced with the `mcp.service` attribute (e.g., `mcp.telemetry`, `mcp.pods`) for granular observability.

3. **Shutdown**:
   - Binary exits when the parent agent process closes the pipe.
   - Flushes OTel telemetry and closes backend connections gracefully.

---

## Environment Variables

| Variable | Example | Purpose |
| :--- | :--- | :--- |
| `THANOS_URL` | `http://localhost:30090` | Metrics via Thanos Query |
| `LOKI_URL` | `http://localhost:30100` | Logs via Loki |
| `TEMPO_URL` | `http://localhost:30200` | Traces via Tempo |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:30317` | Service observability destination |

---

## Troubleshooting

### Connectivity Checks

```bash
# Verify K3s services
kubectl get svc -n observability | grep -E "loki|thanos|tempo"

# Test Telemetry Backends
curl http://localhost:30090/api/v1/query?query=up
curl http://localhost:30100/loki/api/v1/query?query='{job="prometheus"}'
curl http://localhost:30200/api/search
```

### Viewing Gateway Logs

All gateway activity is exported via **OTLP gRPC**.

**Via Loki (Unified View):**

```logql
{service="mcp"}
```

**Via Loki (Domain Specific):**

```logql
{service="mcp", mcp_service="mcp.telemetry"}
```

---

## AI Agent Integration

### Gemini CLI Setup

Add to `~/.gemini/settings.json` under the `mcpServers` key:

```json
{
  "mcpServers": {
    "obs": {
      "command": "/[your path]/mcp_obs_hub",
      "env": {
        "THANOS_URL": "http://localhost:30090",
        "LOKI_URL": "http://localhost:30100",
        "TEMPO_URL": "http://localhost:30200",
        "OTEL_EXPORTER_OTLP_ENDPOINT": "localhost:30317"
      }
    }
  }
}
```

---

## Consolidated Toolset (13 Tools)

| Domain | Key Tools |
| :--- | :--- |
| **Telemetry** | `query_metrics`, `query_logs`, `query_traces`, `investigate_incident` |
| **Kubernetes**| `inspect_pods`, `describe_pod`, `list_pod_events`, `get_pod_logs`, `delete_pod` |
| **Host/Hub** | `hub_inspect_platform`, `hub_inspect_host`, `hub_list_host_services`, `hub_query_service_logs` |
