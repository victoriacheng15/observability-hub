# OpenTelemetry & Observability Guide

## 1. Overview

The Observability Hub uses **OpenTelemetry (OTel)** as the standardized protocol for all signals (Logs, Metrics, and Traces). All telemetry is routed through a central collector in the `observability` namespace and visualized in **Grafana**.

- **Protocol**: OTLP over gRPC.
- **Backend**:
  - **Logs**: Loki
  - **Metrics**: Prometheus
  - **Traces**: Tempo
- **Library**: `pkg/telemetry` (A wrapper around the OTel Go SDK).

---

## 2. Core Philosophy: The "Pure Wrapper"

To maintain scalability and clean separation of concerns, we follow the **Pure Wrapper** pattern:

1. **Library (`pkg/telemetry`, `pkg/db`)**: Owns the "Infrastructure." It handles connection management, standard OTel attributes (like `db.system`), and span lifecycle.
2. **Service (`services/*`)**: Owns the "Domain." It provides the specific business logic, SQL/BSON queries, and schema constants.

---

## 3. Standard Naming Conventions

To ensure dashboard compatibility across the entire fleet, all signals follow these rules:

| Signal Type | Format / Convention | Example |
| :--- | :--- | :--- |
| **Metrics** | `service.entity.action.suffix` | `proxy.webhook.received.total` |
| **Root Spans** | `handler.<name>` (API) or `job.<name>` (Background) | `job.second_brain_sync` |
| **Child Spans** | `<category>.<operation>` | `db.postgres.insert_thought`, `github.fetch` |
| **Attributes** | `entity.property` (Dot notation) | `github.repo`, `db.system` |
| **Log Keys** | `snake_case` for attribute keys | `telemetry.Info("msg", "error_code", 500)` |

---

## 4. Signal Inventory (Grafana Reference)

### 1. Proxy Service

**Metrics:**

- `proxy.webhook.received.total`: Counter
- `proxy.webhook.errors.total`: Counter
- `proxy.webhook.sync.duration.ms`: Histogram
- `proxy.synthetic.request.total`: Counter
- `proxy.synthetic.request.errors.total`: Counter
- `proxy.synthetic.request.duration.ms`: Histogram

**Traces:**

- `handler.webhook`: Root Span
- `webhook.gitops`: Child Span
- `handler.synthetic_trace`: Root Span
- `github.event`, `github.repo`, `github.ref`, `github.action`, `github.merged`: Attributes
- `net.peer.ip`: Attribute
- `webhook.payload_received`: Event

**Logs:**

- `webhook_received`, `webhook_sync_triggered`, `webhook_sync_success`, `webhook_sync_failed`, `webhook_ignored`, `webhook_processed`, `synthetic_trace_payload_received`, `synthetic_trace_processed`

### 2. Reading Sync Service

**Metrics:**

- `reading.sync.processed.total`: Counter
- `reading.sync.errors.total`: Counter
- `reading.sync.duration.ms`: Histogram
- `reading.sync.lag.seconds`: Gauge

**Traces:**

- `job.reading_sync`: Root Span
- `db.postgres.ensure_reading_analytics`: Child Span
- `db.postgres.record_sync_history`: Child Span
- `db.mongodb.fetch_ingested_articles`: Child Span
- `db.mongodb.mark_article_processed`: Child Span
- `db.system`, `db.name`, `db.collection`, `db.query.limit`, `db.mongodb.id`, `db.documents.count`: Attributes
- `db.pool.wait_time`: Attribute

**Logs:**

- `sync_started`, `sync_complete`, `postgres_insert_failed`, `mongo_mark_processed_failed`, `mongo_close_failed`

### 3. Second Brain Service

**Metrics:**

- `second.brain.sync.total`: Counter
- `second.brain.atoms.ingested`: Counter
- `second.brain.token.count.total`: Counter

**Traces:**

- `job.second_brain_sync`: Root Span
- `db.postgres.get_para_stats`: Child Span
- `db.postgres.insert_thought`: Child Span
- `github.fetch`: Child Span
- `ingest.delta`: Iterative Span
- `parse.markdown.duration`: Child Span

**Logs:**

- `database_check_complete`, `ingesting_issue`, `sync_complete`, `sync_skipped`, `fetch_body_failed`, `atom_insert_failed`

### 4. System Metrics Service

**Metrics:**

- `system.metrics.collection.total`: Counter
- `system.metrics.collection.errors`: Counter
- `system.metrics.resource.saturation`: Gauge

**Traces:**

- `job.system_metrics`: Root Span
- `db.postgres.ensure_system_metrics`: Child Span
- `db.postgres.record_metric`: Child Span
- `os.poll_stats`: Child Span
- `os.kernel.version`: Attribute

**Logs:**

- `db_connected`, `collection_complete`, `db_insert_failed`, `otel_init_failed`

### 5. Host Collectors Service

**Metrics:**

- `collectors.collection.total`: Counter
- `collectors.collection.errors`: Counter
- `collectors.tailscale.active`: Gauge

**Traces:**

- `job.collect_batch`: Root Span
- `db.postgres.ensure_system_metrics`: Child Span
- `db.postgres.record_metric`: Child Span
- `host`, `os`, `start`, `end`: Attributes

**Logs:**

- `service_started`, `batch_started`, `batch_complete`, `tailscale_funnel_status`, `tailscale_node_status`, `host_metadata_detection_failed`, `funnel_status_failed`, `tailscale_status_failed`

---

## 5. Operational Commands

### Verifying Signal Flow

**1. Inspect Collector Logs (k3s):**

```bash
kubectl logs -l app.kubernetes.io/name=opentelemetry-collector -n observability
```

**2. Test Local Connectivity:**

```bash
# Ensure the OTLP gRPC port is reachable (standard: 4317)
nc -zv localhost 4317
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
