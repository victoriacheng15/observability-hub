# Observability Architecture

The Observability Hub implements a high-fidelity logging, tracing, and metrics pipeline. The architecture is designed for deep visibility into native host services (via unified logging) and cluster infrastructure (via comprehensive metrics).

## 🛠️ The Unified Pipeline

```mermaid
graph TD
    subgraph Ingestion ["1. Data Ingestion"]
        ScriptLogs[Scripts]
        Proxy[Proxy]
        PromScrapes[Prometheus Scrapes]
        SystemMetrics["System Metrics (Go Custom)"]
    end

    subgraph Processing ["2. Transport & Core Processing"]
        Journal[Systemd Journal]
        Alloy[Alloy]
        OpenTelemetry[OpenTelemetry]
    end

    subgraph Persistence ["3. Data Persistence"]
        subgraph ObservabilityStores ["Observability Stores"]
            Loki[(Loki)]
            Tempo[(Tempo)]
            Prometheus[(Prometheus)]
        end

        subgraph DataStores ["Data Stores"]
            PostgreSQL[(PostgreSQL)]
            MinIO[(MinIO S3)]
        end
    end

    subgraph Visualization ["4. Data Visualization"]
        Grafana[Grafana]
    end
    
    %% Data Flow
    ScriptLogs --> Journal
    Proxy --> OpenTelemetry
    SystemMetrics --> PostgreSQL
    PromScrapes --> ObservabilityStores

    Journal --> Alloy
    Alloy --> ObservabilityStores
    OpenTelemetry --> ObservabilityStores
    Tempo --> ObservabilityStores

    Loki --> MinIO
    Tempo --> MinIO
    Prometheus --> MinIO
    PostgreSQL -->Visualization
    ObservabilityStores --> Visualization
```

## 🪵 Logs

The platform implements a dual-path logging strategy: structured application logs via OpenTelemetry and system-level logs via the host journal.

- **Logging Standards**: To ensure logs are searchable and actionable, all system components must adhere to the **JSON Logging Standard**:

| Field | Description | Example |
| :--- | :--- | :--- |
| `time` | RFC3339 Timestamp | `2026-01-21T22:00:00Z` |
| `level` | Severity (INFO, WARN, ERROR) | `ERROR` |
| `service` | Logic domain name | `proxy` |
| `msg` | Human-readable description | `GitOps sync failed` |
| `repo` | (Optional) Target repository | `mehub` |

- **Collection Pipeline**:
  - **Application Logs**: Services are instrumented with the **OpenTelemetry SDK** to generate logs in OTLP format, sent to the central **OpenTelemetry Collector** via gRPC (NodePort `30317`) or HTTP (NodePort `30318`), which batches and exports them to **Loki**.
  - **System Logs**: Native host services (e.g., `gitops-sync`, `system-metrics`, `tailscale-gate`) write to `stdout`, which `journald` enriches with metadata. **Alloy** (running as a DaemonSet) scrapes `/var/log/journal` directly, filters for these specific units, and pushes to **Loki**.
- **Persistence**:
  - **Loki**: Stores logs with long-term persistence in MinIO S3 buckets (`loki-chunks`, `loki-ruler`, `loki-admin`).

## 📊 Metrics

The platform aggregates infrastructure metrics through Prometheus scraping, application-level metrics via OpenTelemetry, and specialized analytical data.

- **Collection Strategy**:
  - **Infrastructure Scrapes**: **Prometheus** actively pulls metrics from the Kubernetes API, nodes (cAdvisor), pods, service endpoints, and internal exporters (`kube-state-metrics`, `node-exporter`).
  - **Telemetry Ingestion**: Prometheus is configured with the `remote-write-receiver` enabled to ingest OTLP metrics from the **OpenTelemetry Collector** and derived span-metrics (e.g., service graphs) from **Tempo**.
  - **Specialized Analytics**: Custom Go services (`system-metrics`) write specialized host-level time-series data directly to **PostgreSQL** for analytical reporting.
- **Persistence**:
  - **Local Storage**: Prometheus maintains a high-resolution 24-hour local TSDB on `local-path` persistent volumes.
  - **Long-term Retention**: The **Thanos** sidecar seamlessly offloads TSDB blocks to MinIO S3 (`prometheus-blocks`) for infinite metrics retention and historical analysis.

## 🔭 Traces

Distributed tracing is powered by OpenTelemetry for correlation and performance profiling across high-throughput pipelines.

- **Collection Pipeline**:
  - **Instrumentation**: Services use the **OpenTelemetry SDK** to generate spans in OTLP format.
  - **Ingestion**: Spans are sent to the **OpenTelemetry Collector** via gRPC (NodePort `30317`) or HTTP (NodePort `30318`), which batches and exports them to **Grafana Tempo**.
  - **Processing**: Tempo analyzes raw spans to generate derived **Service Graphs** and **Span Metrics**, which are pushed to Prometheus via `remote_write` for operational correlation.
- **Persistence**:
  - **Tempo**: Stores traces with long-term persistence in MinIO S3 buckets (`tempo-traces`).

## 🗄️ Shared Data Stores

- **PostgreSQL**: Stores analytical metrics and specialized time-series data using local persistent volumes.
- **MinIO S3**: Provides unified object storage for Loki logs, Tempo traces, and Prometheus/Thanos metrics blocks.

Access is secured via internal Kubernetes networking (`minio.observability.svc.cluster.local:9000`) and managed via specialized secrets (`minio-thanos-secret`, etc.).
