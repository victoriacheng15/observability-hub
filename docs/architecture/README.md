# Observability Hub Architecture

This document serves as the entry point for the system's architecture.

## System Context

The hub integrates standard observability tools with custom Go services.

```mermaid
graph TD
    subgraph "External Sources"
        Apps[Client Apps]
        Mongo[(MongoDB Atlas)]
    end

    subgraph "Host Environment"
        Hardware[Host Hardware]
        HostServices[Host Systemd Services]
    end

    subgraph "Observability Hub"
        direction TB
        
        subgraph "Collection & ETL"
            Metrics[System Metrics Collector]
            Sync[Reading Sync Service]
            Proxy[Proxy Service / ETL]
            Promtail[Promtail]
        end

        subgraph "Storage"
            PG[(PostgreSQL)]
            Loki[(Loki)]
        end

        subgraph "Visualization"
            Grafana[Grafana]
        end
    end

    %% Application Data Flow
    Apps -->|Events| Mongo
    Sync -->|Triggers ETL| Proxy
    Mongo -->|Reads Events| Proxy
    Proxy -->|Writes Structured Data| PG

    %% System Metrics Flow
    Hardware -->|Stats: CPU, RAM, Disk, Net| Metrics
    Metrics -->|Writes Metrics| PG

    %% GitOps & Automation
    HostServices -->|Manages| Metrics
    HostServices -->|Manages| Sync

    %% Logging Flow
    Proxy -.->|Docker Logs| Promtail
    HostServices -.->|Systemd Logs| Promtail
    Metrics -.->|Systemd Logs| Promtail
    Promtail -->|Pushes Logs| Loki

    %% Visualization
    PG -->|Query Data| Grafana
    Loki -->|Query Logs| Grafana
```

## Detailed Architecture Documents

| Component | Description |
| :----------- | :------------- |
| **[Proxy Service](./proxy-service.md)** | Architecture of the Go-based API Gateway and ETL Engine. Bridges external data (MongoDB) with PostgreSQL via triggered sync. |
| **[System Metrics](./system-metrics.md)** | Details on the custom host telemetry collector (`gopsutil`). Pushes data directly to the `system_metrics` table in PostgreSQL (TimescaleDB). |
| **[Infrastructure](./infrastructure.md)** | Deployment (Docker), Storage (Postgres/Loki), and Security config. |
| **[Systemd Services](./systemd-services.md)** | Automation architecture for GitOps, ETL triggers, and telemetry using systemd units and timers. |
| **[GitOps Reconciliation](./../decisions/005-gitops-reconciliation-engine.md)** | Systemd-driven agent for automated, self-healing repository synchronization. |

## Related Documentation

- **[Decisions](../decisions/)**: Architecture Decision Records (ADRs).
- **[Planning](../planning/)**: Future trends and RFC drafts.
