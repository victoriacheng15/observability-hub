# MCP Telemetry Service: Standalone Execution Guide

The `mcp-telemetry` service bridges the host tier to the LGTM stack (Thanos, Loki, Tempo) via the Model Context Protocol, enabling AI agents to query metrics, logs, and traces autonomously.

---

## Quick Commands

### Build (Development)

```bash
make mcp-telemetry-build
```

This command:
1. Compiles `cmd/mcp-telemetry` → `bin/mcp_telemetry`

### Run Manually

```bash
./bin/mcp_telemetry
```

Operational logs are emitted to **stderr** to ensure `stdout` remains dedicated to the JSON-RPC protocol required by MCP.

---

## How It Works

### Standalone Architecture

```text
AI Agent (Gemini CLI / Copilot)
└── Spawn Binary (bin/mcp_telemetry)
    └── Persistent connections to:
        ├── Thanos @ localhost:30090 (Prometheus query)
        ├── Loki @ localhost:30100 (Log queries)
        └── Tempo @ localhost:30200 (Trace queries)
```

### Service Lifecycle

1. **Initialization**:
   - Reads `.env` (loads `THANOS_URL`, `LOKI_URL`, `TEMPO_URL`)
   - Initializes OTel telemetry (traces, metrics, logs → OTLP gRPC @ localhost:30317)
   - Establishes HTTP client pool to all 3 backends

2. **Execution**:
   - Listens for MCP tool calls over `stdin`.
   - Executes queries against Thanos/Loki/Tempo.
   - Returns structured results over `stdout`.
   - **Logs all activity via OTel & stderr**.

3. **Shutdown**:
   - Binary exits when the parent agent process closes the pipe.
   - Flushes OTel telemetry and closes connections gracefully.

---

## Environment Variables

All are loaded from `.env` at startup:

| Variable | Example | Purpose |
| :--- | :--- | :--- |
| `THANOS_URL` | `http://localhost:30090` | Prometheus-compatible queries |
| `LOKI_URL` | `http://localhost:30100` | LogQL queries |
| `TEMPO_URL` | `http://localhost:30200` | Trace retrieval |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:30317` | Service observability destination |

---

## Troubleshooting

### Connectivity refused

```bash
# Verify K3s services are exposing NodePorts
kubectl get svc -n observability | grep -E "loki|thanos|tempo"

# Test connectivity
curl http://localhost:30090/api/v1/query?query=up
curl http://localhost:30100/loki/api/v1/query?query='{job="prometheus"}'
curl http://localhost:30200/api/search
```

### Viewing Service Logs

All logs from mcp-telemetry are exported via **OTLP gRPC** to Loki and Tempo. Operational output is also mirrored to `stderr`.

**Via Loki Dashboard:**
```logql
{service="mcp.telemetry"}
```

**Via Tempo Traces:**
1. Open Tempo in Grafana
2. Search by service: `mcp.telemetry`
3. View logs attached to trace spans

---

## AI Agent Integration

### Gemini CLI Setup

Add to `~/.gemini/settings.json` under the `mcpServers` key:

```json
{
  "mcpServers": {
    "mcp-telemetry": {
      "command": "/home/[user]/software/observability-hub/bin/mcp_telemetry",
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

### GitHub Copilot CLI Setup

Add to `~/.copilot/mcp-config.json`:

```json
{
  "mcpServers": {
    "mcp-telemetry": {
      "command": "/home/[user]/software/observability-hub/bin/mcp_telemetry",
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

### Available Tools

| Tool | Purpose |
| :--- | :--- |
| `query_metrics` | Execute PromQL against Thanos |
| `query_logs` | Execute LogQL against Loki |
| `query_traces` | Fetch/Search traces in Tempo |
| `investigate_incident` | Cross-correlate LGTM signals |
