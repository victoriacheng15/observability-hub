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

**Service Name:** `proxy` | **Tracer/Meter:** `proxy.synthetic`

**Metrics:**

- `proxy.synthetic.request.total`: Counter
- `proxy.synthetic.request.errors.total`: Counter
- `proxy.synthetic.request.duration.ms`: Histogram

**Traces:**

- `HTTP <method> <path>`: Root Span (via `otelhttp`)
- `handler.synthetic_trace`: Process Span
- **Attributes:** `app.synthetic_id`, `app.traffic_mode`, `app.business.region`, `app.business.timezone`, `app.business.device`, `app.business.network_type`, `app.latency_target_ms`, `error`, `error.message`
- **Events:** `request.payload.received`, `request.payload.decode_failed`, `processing.simulated_delay`

**Logs:**

- `otel_telemetry_enabled`, `🚀 The GO proxy listening on port`, `request_processed`, `synthetic_trace_payload_received`, `synthetic_trace_payload_decode_failed`, `synthetic_trace_response_encode_failed`, `synthetic_trace_processed`

### 2. Ingestion Service

**Service Name:** `ingestion` | **Tracer:** `ingestion.engine`

**Root Spans:**

- `task.reading`: Root for Reading Sync
- `task.brain`: Root for Brain Sync

#### 2.1 Reading Task

**Tracer/Meter:** `reading.sync`

**Metrics:**

- `reading.sync.total`: Counter (Sync Runs)
- `reading.sync.processed.total`: Counter (Documents, Global Scope)
- `reading.sync.errors.total`: Counter (Errors during sync)
- `reading.sync.duration.ms`: Histogram (Sync latency)
- `reading.sync.lag.seconds`: Gauge (Time since last success)

**Traces:**

- `job.reading_sync`: Process Span
- `db.postgres.ensure_reading_analytics`: Child Span
- `db.postgres.ensure_reading_sync_history`: Child Span
- `db.postgres.record_sync_history`: Child Span
- `db.postgres.insert_reading_analytics`: Child Span
- `db.mongodb.fetch_ingested_articles`: Child Span
- `db.mongodb.mark_article_processed`: Child Span
- **Attributes:** `task.name`, `db.documents.count`

**Logs:**

- `sync_started`, `failed_to_record_sync_history`, `postgres_insert_failed`, `mongo_mark_processed_failed`, `sync_complete`, `mongo_close_failed`

#### 2.2 Brain Task

**Tracer/Meter:** `second.brain`

**Metrics:**

- `second.brain.sync.total`: Counter (Sync Runs)
- `second.brain.sync.processed.total`: Counter (Thoughts, Global Scope)
- `second.brain.sync.errors.total`: Counter (Errors during sync)
- `second.brain.sync.duration.ms`: Histogram (Sync latency)
- `second.brain.sync.lag.seconds`: Gauge (Time since last success)
- `second.brain.token.count.total`: Counter (Tokens, Global Scope)

**Traces:**

- `job.second_brain_sync`: Process Span
- `github.fetch`: Child Span
- `ingest.delta`: Child Span
- `parse.markdown.duration`: Child Span
- `db.postgres.ensure_second_brain`: Child Span
- `db.postgres.ensure_brain_sync_history`: Child Span
- `db.postgres.record_brain_sync_history`: Child Span
- `db.postgres.get_latest_entry_date`: Child Span
- `db.postgres.insert_thought`: Child Span
- **Attributes:** `github.issue_number`, `issue.title`, `atoms.count`

**Logs:**

- `sync_started`, `database_check_complete`, `sync_skipped`, `ingesting_issue`, `fetch_body_failed`, `atom_insert_failed`, `failed_to_record_brain_sync_history`, `sync_complete`

### 3. Analytics Service (Host Telemetry)

**Service Name:** `analytics` | **Tracer/Meter:** `analytics`

**Metrics:**

- `analytics.collection.total`: Counter
- `analytics.collection.errors`: Counter
- `analytics.tailscale.active`: Gauge

**Traces:**

- `job.collect_batch`: Root Span
- `db.postgres.ensure_system_metrics`: Child Span
- `db.postgres.create_hypertable`: Child Span
- `db.postgres.record_metric`: Child Span
- **Attributes:** `host`, `os`, `start`, `end`

**Logs:**

- `service_started`, `batch_started`, `batch_complete`, `tailscale_funnel_status`, `tailscale_node_status`, `host_metadata_detection_failed`, `funnel_status_failed`, `tailscale_status_failed`, `service_shutting_down`

---

## 5. Operational Commands

### Verifying Signal Flow

**1. Inspect Collector Logs (k3s):**

```bash
kubectl logs -l app.kubernetes.io/name=opentelemetry-collector -n observability
```

**2. Test Local Connectivity:**

```bash
# Ensure the OTLP gRPC NodePort is reachable (30317)
nc -zv localhost 30317
```

**3. Check Health Extension:**

```bash
# The collector exposes a health check on port 13133
curl http://localhost:13133/
```

### Adding a New Signal

Always use the `telemetry` package to ensure automatic resource tagging (`host.name`, `service.name`).

```go
import "telemetry"

// 1. Logs
telemetry.Info("process_started", "item_id", id)

// 2. Metrics (Counter)
counter, _ := telemetry.NewInt64Counter(meter, "my.service.total", "Description")
telemetry.AddInt64Counter(ctx, counter, 1)

// 3. Traces
ctx, span := tracer.Start(ctx, "job.my_task")
defer span.End()
```
