# OpenTelemetry & Observability Guide

## 1. Overview

The Observability Hub uses **OpenTelemetry (OTel)** as the standardized protocol for all signals (Logs, Metrics, and Traces). All telemetry is routed through a central collector in the `observability` namespace and visualized in **Grafana**.

- **Protocol**: OTLP over gRPC.
- **Backend**:
  - **Logs**: Loki (via OTLP)
  - **Metrics**: Prometheus (via Prometheus Remote Write)
  - **Traces**: Tempo (via OTLP)
- **Library**: `internal/telemetry` (A wrapper around the OTel Go SDK).
- **Collector Endpoint**:
  - **Host Services**: `localhost:30317` (NodePort)
  - **K3s Services**: `opentelemetry.observability.svc.cluster.local:4317`

---

## 2. Core Philosophy: The "Pure Wrapper"

To maintain scalability and clean separation of concerns, we follow the **Pure Wrapper** pattern:

1. **Library (`internal/telemetry`, `internal/db`)**: Owns the "Infrastructure." It handles connection management, standard OTel attributes (like `db.system`), and span lifecycle.
2. **Service (`cmd/*`)**: Owns the "Domain." It provides the specific business logic, SQL/BSON queries, and schema constants.

---

## 3. Standard Naming Conventions

To ensure dashboard compatibility across the entire fleet, all signals follow these rules:

| Signal Type | Format / Convention | Example |
| :--- | :--- | :--- |
| **Metrics** | `service.entity.action.suffix` | `proxy.webhook.received.total` |
| **Root Spans** | `handler.<name>` (API) or `job.<name>` (Background) | `job.reading_sync` |
| **Child Spans** | `<category>.<operation>` | `db.postgres.insert_thought`, `github.fetch` |
| **Attributes** | `entity.property` (Dot notation) | `github.repo`, `db.system` |
| **Log Keys** | `snake_case` for attribute keys | `telemetry.Info("msg", "error_code", 500)` |

---

## 4. Signal Inventory (Grafana Reference)

### 1. Proxy Service

**Service Name:** `proxy` | **Tracers:** `proxy.synthetic`, `proxy.webhook`, `proxy/home`

**Metrics:**

- `proxy.synthetic.request.total`: Counter (labeled by `app.traffic_mode`)
- `proxy.synthetic.request.errors.total`: Counter
- `proxy.synthetic.request.duration.ms`: Histogram
- `proxy.webhook.received.total`: Counter (labeled by `github.event`)
- `proxy.webhook.errors.total`: Counter
- `proxy.webhook.sync.duration.ms`: Histogram

**Traces:**

- `HTTP <method> <path>`: Root Span (via `otelhttp`)
- `handler.synthetic_trace`: Process Span
- `handler.webhook`: Webhook entry point
- `webhook.gitops`: Async GitOps reconciliation execution
- **Attributes:** `app.synthetic_id`, `app.traffic_mode`, `app.business.region`, `app.business.timezone`, `app.business.device`, `app.business.network_type`, `app.latency_target_ms`, `github.event`, `github.repo`, `github.ref`, `github.action`, `github.merged`, `net.peer.ip`
- **Events:** `request.payload.received`, `processing.simulated_delay`, `outbound.response_received`

**Logs:**

- `request_processed` (Standard Access Log), `webhook_received`, `webhook_sync_triggered`, `webhook_sync_success`, `synthetic_trace_payload_received`, `synthetic_trace_processed`

### 2. Ingestion Service

**Service Name:** `ingestion` | **Tracers:** `reading.sync`, `second.brain`, `ingestion.engine`

**Root Spans (Engine):**

- `task.reading`: Root for Reading Sync
- `task.brain`: Root for Brain Sync

#### 2.1 Reading Task (`reading.sync`)

**Metrics:**

- `reading.sync.total`: Counter
- `reading.sync.processed.total`: Counter (Total articles processed)
- `reading.sync.errors.total`: Counter
- `reading.sync.duration.ms`: Histogram
- `reading.sync.lag.seconds`: Gauge (Time since last success)

**Traces:**

- `job.reading_sync`: Process Span
- `db.mongodb.fetch_ingested_articles`: Child Span
- `db.mongodb.mark_article_processed`: Child Span
- **Attributes:** `task.name`, `db.documents.count`

#### 2.2 Brain Task (`second.brain`)

**Metrics:**

- `second.brain.sync.total`: Counter
- `second.brain.sync.processed.total`: Counter (Total thoughts processed)
- `second.brain.sync.errors.total`: Counter
- `second.brain.sync.duration.ms`: Histogram
- `second.brain.sync.lag.seconds`: Gauge
- `second.brain.token.count.total`: Counter (Token count for LLM context)

**Traces:**

- `job.second_brain_sync`: Process Span
- `github.fetch`: GitHub CLI interaction
- `ingest.delta`: Single issue processing
- `parse.markdown.duration`: Atomization logic
- **Attributes:** `github.issue_number`, `issue.title`, `atoms.count`

### 3. Analytics Service (Host Telemetry)

**Service Name:** `analytics` | **Tracer/Meter:** `analytics`

**Metrics:**

- `analytics.batch.total`: Counter (Labeled by `host`, `os`)
- `analytics.batch.errors.total`: Counter
- `analytics.tailscale.active`: Observable Gauge (1 = Active, 0 = Inactive)

**Traces:**

- `job.collect_batch`: Root Span
- **Attributes:** `host`, `os`, `start`, `end` (RFC3339)

**Logs:**

- `service_started`, `batch_started`, `feature_analytics_recorded`, `value_unit_recorded`, `tailscale_funnel_status`, `batch_complete`

### 4. Database Wrappers (Internal Library)

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

### 5. MCP Agents (Unified Gateway)

**Service Name:** `mcp` (Binary: `mcp_obs_hub`) | **Tracer/Meter:** `mcp`

**Metrics:**

- `mcp_tool_calls_total`: Counter (labeled by `tool`, `service`, `status`)
- `mcp_tool_duration_ms`: Histogram (labeled by `tool`, `service`)

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
