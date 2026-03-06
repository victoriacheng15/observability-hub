# MCP Telemetry Service: Rebuild & Restart Guide

The `mcp-telemetry` service bridges the host tier to the LGTM stack (Thanos, Loki, Tempo) via the Model Context Protocol, enabling AI agents to query metrics, logs, and traces autonomously.

---

## Quick Commands

### Build & Restart (Development)

```bash
make mcp-telemetry-build
```

This single command:

1. Compiles `cmd/mcp-telemetry` → `bin/mcp_telemetry`
2. Automatically restarts the systemd service
3. Outputs logs immediately

### Manual Restart (No Rebuild)

```bash
sudo systemctl restart mcp-telemetry.service
```

### Check Service Status

```bash
systemctl status mcp-telemetry.service
```

### Follow Logs in Real-Time

```bash
journalctl -u mcp-telemetry.service -f
```

---

## How It Works

### Deployment Architecture

```
Host Tier (Systemd)
├── mcp-telemetry.service (Long-running server)
│   └─ Persistent connections to:
│       ├── Thanos @ localhost:30090 (Prometheus query)
│       ├── Loki @ localhost:30100 (Log queries)
│       └── Tempo @ localhost:30200 (Trace queries)
```

### Service Lifecycle

1. **Start** (`systemctl start`):
   - Reads `.env` (loads `THANOS_URL`, `LOKI_URL`, `TEMPO_URL`)
   - Initializes OTel telemetry (traces, metrics, logs → OTLP gRPC @ localhost:30317)
   - Establishes HTTP client pool to all 3 backends
   - Blocks on signal channel → waits for connections/shutdown

2. **Running**:
   - Accepts MCP tool calls (e.g., `query_metrics`, `query_logs`, etc.)
   - Maintains persistent connections to Thanos/Loki/Tempo
   - Routes queries, validates input (SQL injection prevention)
   - **Logs all activity via OTel** → exported to Loki & Tempo (not journald)

3. **Stop** (`systemctl stop` or `Ctrl+C`):
   - Receives SIGTERM signal
   - Gracefully closes HTTP connections
   - Flushes OTel telemetry
   - Exits cleanly

4. **Restart** (`systemctl restart`):
   - Stops the service gracefully
   - Rebuilds connection pools (fresh state)
   - Restarts listening

---

## Environment Variables

All are loaded from `.env` at service startup:

| Variable | Example | Purpose |
| :--- | :--- | :--- |
| `THANOS_URL` | `http://localhost:30090` | Prometheus-compatible queries |
| `LOKI_URL` | `http://localhost:30100` | LogQL queries |
| `TEMPO_URL` | `http://localhost:30200` | Trace retrieval |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:30317` | Service observability destination |

---

## Troubleshooting

### Service fails to start

**Binary errors** (when running directly):

```bash
./bin/mcp_telemetry 2>&1 | grep -E "ERROR|WARN"
```

**Systemd service errors** (when running as service):

```bash
systemctl status mcp-telemetry.service
# Service logs are exported to Loki, not journalctl
# View via: {service="mcp.telemetry"} in Loki UI
```

Check for:

- Missing `THANOS_URL` in `.env`
- K3s services not running (check `make k3s-status`)
- Port conflicts (verify NodePorts are exposed)

### Slow queries

- Check Thanos/Loki/Tempo health: `curl http://localhost:30090/-/healthy`
- Verify network latency to K3s: `time curl http://localhost:30090/api/v1/query?query=up`
- Check **service logs in Loki** for performance issues:

  ```
  {service="mcp.telemetry"} | logfmt
  ```

### Viewing Service Logs

All logs from mcp-telemetry are exported via **OTLP gRPC** to Loki and Tempo. Do NOT use `journalctl`.

**Via Loki Dashboard:**

```
{service="mcp.telemetry"}
```

**Via Grafana Logs:**

1. Open Grafana → Explore → Loki
2. Query: `{service="mcp.telemetry"} | logfmt`
3. View structured logs with levels (INFO, WARN, ERROR)

**Via Tempo Traces:**

1. Open Tempo in Grafana
2. Search by service: `mcp.telemetry`
3. View logs attached to trace spans

### Connection refused

```bash
# Verify K3s services are exposing NodePorts
kubectl get svc -n observability | grep -E "loki|thanos|tempo"

# Test connectivity
curl http://localhost:30090/api/v1/query?query=up
curl http://localhost:30100/loki/api/v1/query?query='{job="prometheus"}'
curl http://localhost:30200/api/search
```

---

## Healthy Service State

```bash
$ systemctl status mcp-telemetry.service
● mcp-telemetry.service - MCP Telemetry Server
     Loaded: loaded (/etc/systemd/system/mcp-telemetry.service; enabled; vendor preset: enabled)
     Active: active (running) since 2026-03-05 21:24:00 UTC; 1min ago
   Main PID: 12345 (mcp_telemetry)
      Tasks: 5 (limit: 4915)
     Memory: 15.2M
```

✅ **Healthy indicators:**

- `Active: active (running)`
- `Main PID: <non-zero>`
- Memory usage stable (typically 10-50MB)
- No recent restarts
- Logs appear in Loki with `service="mcp.telemetry"`

---

## Installation (One-Time)

If service isn't installed yet:

```bash
make install-services    # Symlinks systemd/ units to /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now mcp-telemetry.service
```

---

## Integration with Agents

Agents connect to `mcp-telemetry` via stdio or direct process invocation:

```bash
# Agent calls the service for metrics query
./bin/mcp_telemetry --tool query_metrics --query "rate(http_requests_total[5m])"
```

Or via systemd socket activation (future enhancement).

---

## Monitoring

Track service health in your observability stack:

**Via Loki (Logs):**

```
{service="mcp.telemetry"}
```

**Via Tempo (Traces):**

- Open Grafana → Tempo
- Search by service name: `mcp.telemetry`
- View distributed traces with embedded logs

**Metrics about mcp-telemetry:**

- Query Thanos for service performance
- Check OTLP gRPC export metrics: `{exporter="otlpgrpc"}`

---

## GitHub Copilot & Google Gemini CLI Integration

The mcp-telemetry server exposes MCP tools that can be called by GitHub Copilot CLI and Google's Gemini CLI.

### GitHub Copilot CLI Setup

Add to `~/.copilot/mcp-config.json`:

```json
{
  "mcpServers": {
    "mcp-telemetry": {
      "command": "/home/[user]/software/observability-hub/bin/mcp_telemetry",
      "args": [],
      "tools": ["*"],
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

Verify with `/mcp` inside a Copilot CLI session.

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

Restart Gemini CLI after saving.

### Available Tools

| Tool | Status | Input | Example Query |
| :--- | :--- | :--- | :--- |
| `query_metrics` | ✅ Implemented | `query: string` | `up{job="prometheus"}` |
| `query_logs` | ⏳ No-op | `query: string` | `{level="error"}` |
| `query_traces` | ⏳ No-op | `trace_id: string` | `4bf92f3577b34da6a3ce929d0e0e4736` |

### Test Query

Ask Copilot/Gemini:

```
Query metrics to get the 95th percentile request latency:
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

The AI will call `query_metrics` and return results from Thanos.

### Troubleshooting

- **Binary not found**: Verify path and run `chmod +x bin/mcp_telemetry`
- **Connection refused**: Ensure mcp-telemetry is running (`systemctl status mcp-telemetry.service`)
- **Tool timeout**: Check K3s NodePort health (`curl http://localhost:30090/-/healthy`)
