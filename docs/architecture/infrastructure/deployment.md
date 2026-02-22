# Infrastructure & Deployment

The infrastructure layer follows a **hybrid model**: core data services (Storage, Logs, Viz) are orchestrated via **Kubernetes (k3s)**, while application logic and automation agents run as native host-level Systemd services for direct hardware and filesystem access.

## Component Details

### ☸️ Data Infrastructure (Kubernetes)

| Component | Role | Details |
| :--- | :--- | :--- |
| **Collectors** | Host Telemetry Collector | DaemonSet for collecting host telemetry and gathering Tailscale status. |
| **Grafana** | Visualization | Deployment for unified dashboarding UI. |
| **Loki** | Log Aggregation | StatefulSet for indexing metadata-tagged logs. |
| **MinIO** | Object Storage | Deployment for S3-compatible storage, serving as backup for Prometheus, Loki, and Tempo. |
| **OpenTelemetry Collector** | Telemetry Hub | Deployment for receiving and processing traces, metrics, and logs. |
| **PostgreSQL** | Primary Storage | StatefulSet with TimescaleDB + PostGIS for metrics and analytical data. |
| **Prometheus** | Metrics Storage | Deployment for time-series infrastructure and service metrics. |
| **Tempo** | Trace Storage | StatefulSet for high-scale distributed tracing persistence via MinIO. |
| **Thanos** | Long-term Metrics | StatefulSet for querying historical metrics stored in MinIO. |

### 🚀 Core Services (Native Go)

| Component | Role | Details |
| :--- | :--- | :--- |
| **Proxy Service** | API Gateway | Handles webhooks, GitOps triggers, and Data Pipelines. |
| **Metrics Collector** | Telemetry Agent | Collects host hardware statistics (CPU, RAM, Disk). |
| **Second Brain** | Knowledge Ingest | Ingests atomic journal entries from GitHub Issues. |

### 🛠️ Automation & Security (Native Script)

| Component | Role | Details |
| :--- | :--- | :--- |
| **OpenBao** | Secret Store | Centralized, encrypted management for sensitive config. |
| **Tailscale Gate** | Security Agent | Manages public funnel access based on service health. |
| **Reading Sync** | Data Pipeline | Timer-triggered task to sync cloud data to local storage. |

## Data Flow: Unified Observability

```mermaid
sequenceDiagram
    participant App as Go Services
    participant Script as Bash Scripts
    participant Collectors_Agent as Collectors
    participant OTel_Collector as OpenTelemetry Collector
    participant Observability as Observability (Loki, Prometheus, Tempo)
    participant Grafana as Grafana

    App->>OTel_Collector: Logs, Metrics, Traces (OTLP)
    Script->>OTel_Collector: Logs, Metrics, Traces (OTLP)
    Collectors_Agent->>OTel_Collector: Metrics, Traces (OTLP)

    OTel_Collector->>Observability: Push Logs, Metrics, Traces
    Observability->>Grafana: Query Logs, Metrics, Traces
```

## Deployment Strategy

- **Orchestration**: **Kubernetes (k3s)** for data infrastructure.
- **Native Services**: Systemd units for high-performance and host-level tasks.
- **Automation**: `Makefile` for lifecycle management (build, restart, install).
- **Persistence**: Kubernetes **PersistentVolumeClaims (PVCs)** for data durability.
- **Event-Driven Sync**: GitHub Webhooks trigger the local `gitops_sync.sh` via the Proxy.

## Configuration & Security

### Network Security

- **Isolation**: Workloads communicate on an internal Kubernetes cluster network.
- **Funnel Integration**: The **Tailscale Gate** manages `tailscale funnel` to expose only port `8085` (Proxy) to the public internet securely via port `8443`.
- **Exposed Ports**:
  - `30000`: Grafana (Kubernetes NodePort)
  - `30432`: PostgreSQL (Kubernetes NodePort)
  - `30317`: OpenTelemetry (OTLP gRPC NodePort)
  - `30318`: OpenTelemetry (OTLP HTTP NodePort)
  - `8085`: Proxy Service (Publicly available via Tailscale Funnel)
