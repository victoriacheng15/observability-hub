# OpenTelemetry & Observability Guide

## Overview

The Observability Hub uses **OpenTelemetry (OTel)** as the standardized protocol for all signals (Logs, Metrics, and Traces). All telemetry is routed through a central collector in the `observability` namespace and visualized in **Grafana**.

- **Protocol**: OTLP over gRPC.
- **Backend**:
  - **Logs**: Loki (via OTLP)
  - **Metrics**: Prometheus (via Prometheus Remote Write)
  - **Traces**: Tempo (via OTLP)
- **Library**: `internal/telemetry` (A wrapper around the OTel Go SDK).
- **Collector Endpoint**:
  - **Host Services (Proxy, Gate)**: `localhost:30317` (NodePort)
  - **K3s Services (Worker, Simulation)**: `opentelemetry.observability.svc.cluster.local:4317`

---

## Core Philosophy: The "Pure Wrapper"

To maintain scalability and clean separation of concerns, we follow the **Pure Wrapper** pattern:

1. **Library (`internal/telemetry`, `internal/db`)**: Owns the "Infrastructure." It handles connection management, standard OTel attributes (like `db.system`), and span lifecycle.
2. **Service (`cmd/*`)**: Owns the "Domain." It provides the specific business logic, SQL/BSON queries, and schema constants.

---

## Standard Naming Conventions

To ensure dashboard compatibility across the entire fleet, all signals follow these rules:

| Signal Type | Format / Convention | Example |
| :--- | :--- | :--- |
| **Metrics** | `service.entity.action.suffix` | `proxy.webhook.received.total` |
| **Root Spans** | `handler.<name>` (API) or `job.<name>` (Background) | `job.reading_sync` |
| **Child Spans** | `<category>.<operation>` | `db.postgres.insert_thought`, `github.fetch` |
| **Attributes** | `entity.property` (Dot notation) | `github.repo`, `db.system` |
| **Log Keys** | `snake_case` for attribute keys | `telemetry.Info("msg", "error_code", 500)` |

---

## Signal Inventory (Grafana Reference)

### Proxy Service

**Service Name:** `proxy` | **Tracers:** `proxy.synthetic`, `proxy.webhook`, `proxy/home`

**Metrics:**

- `proxy.synthetic.request.total`: Counter (labeled by `app.traffic_mode`)
- `proxy.synthetic.request.errors.total`: Counter
- `proxy.synthetic.request.duration`: Histogram
- `proxy.webhook.received.total`: Counter (labeled by `github.event`)
- `proxy.webhook.errors.total`: Counter
- `proxy.webhook.sync.duration`: Histogram

**Traces:**

- `HTTP <method> <path>`: Root Span (via `otelhttp`)
- `handler.synthetic_trace`: Process Span
- `handler.webhook`: Webhook entry point
- `webhook.gitops`: Async GitOps reconciliation execution
- **Attributes:** `app.synthetic_id`, `app.traffic_mode`, `app.business.region`, `app.business.timezone`, `app.business.device`, `app.business.network_type`, `app.latency_target_ms`, `github.event`, `github.repo`, `github.ref`, `github.action`, `github.merged`, `net.peer.ip`
- **Events:** `request.payload.received`, `processing.simulated_delay`, `outbound.response_received`

**Logs:**

- `request_processed` (Standard Access Log), `webhook_received`, `webhook_sync_triggered`, `webhook_sync_success`, `synthetic_trace_payload_received`, `synthetic_trace_processed`

### Unified Worker (CronJob)

**Service Names:** `worker.analytics`, `worker.ingestion` | **Tracers:** `worker.analytics`, `worker.ingestion`, `worker.run`

**Metrics (Emitted):**
- `worker.batch.total`: Counter (Total worker invocations)
- `worker.batch.errors.total`: Counter (Total execution failures)
- `worker.batch.duration`: Histogram (Execution time per run in ms)

**Mode-Specific Metrics:**
- `worker.brain.sync.total`: Counter (Ingestion mode)
- `worker.reading.sync.total`: Counter (Ingestion mode)
- `worker.tailscale.active`: Observable Gauge (Analytics mode; 1 = Active)

**Traces:**
- `worker.run`: Root Span for every execution.
- `db.postgres.*`, `github.fetch`, `thanos.query`: Mode-specific child spans.

**Logs:**

- **Shared:** `postgres_initialization_failed`, `schema_initialization_failed`
- **Analytics Mode:** `analytics_batch_starting`, `analytics_factors_loaded`, `analytics_batch_complete`
- **Ingestion Mode:** `starting_ingestion_tasks`, `running_task`, `task_succeeded`, `task_failed`, `all_ingestion_tasks_finished`, `failed_to_record_brain_sync_history`, `failed_to_record_reading_sync_history`

### Database Wrappers (Internal Library)

**Tracers:** `db/postgres`, `db/mongodb`

**Common Attributes:**

- `db.system`: `postgresql` or `mongodb`
- `db.statement`: The SQL query or BSON filter (redacted/marshaled)
- `db.user`: Current database user
- `db.name`: Database name

**Postgres Specific:**

- `db.pool.wait_time`: 64-bit int of pool wait duration

**MongoDB Specific:**

- `db.collection`: Target collection
- `db.query.limit`: Operation limit
- `db.mongodb.id`: Hex string of the target document ID

### MCP Agents (Unified Gateway)

**Service Name:** `mcp` (Binary: `mcp_obs_hub`) | **Tracer/Meter:** `mcp`

**Metrics:**

- `mcp_tool_calls_total`: Counter (labeled by `tool`, `service`, `status`)
- `mcp_tool_duration`: Histogram (labeled by `tool`, `service`)

**Traces:**

- `mcp.tool.<name>`: Root Span for each tool execution (e.g., `mcp.tool.inspect_pods`)
- **Attributes:**
  - `mcp.tool`: The specific tool name.
  - `mcp.service`: The domain discriminator (`mcp.telemetry`, `mcp.pods`, or `mcp.hub`).

**Logging Pattern:**
The unified agent emits logs for sequential provider initialization and execution visibility:

- `registered hub tools (mcp.hub)`, `registered pods tools (mcp.pods)`, `registered telemetry tools (mcp.telemetry)`
- `executing PromQL query`, `executing LogQL query`, `retrieving trace from Tempo`
- `investigating incident` (Macro-tool orchestration)
- `using local kubeconfig` (Diagnostic logs for the Pods provider)
