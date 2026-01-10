# Observability Hub Architecture

This document serves as the entry point for the system's architecture.

## System Context

The hub integrates standard observability tools with custom Go services.

```mermaid
graph TD
    subgraph "Host Environment"
        Hardware[Host Hardware]
        Docker[Docker Containers]
        GitOpsAgent[GitOps Reconciliation Agent]
    end

    subgraph "External Data"
        Mongo[(MongoDB)]
    end

    subgraph "Observability Hub"
        direction TB
        
        subgraph "Collection"
            Metrics[System Metrics Collector]
            Promtail[Promtail Agent]
            Proxy[Proxy Service / ETL]
        end

        subgraph "Storage"
            PG[(PostgreSQL)]
            Loki[(Loki)]
        end

        subgraph "Visualization"
            Grafana[Grafana Dashboard]
        end
    end

    %% Data Flows
    Hardware -->|Stats: CPU, RAM, Disk, Memory| Metrics
    Metrics -->|Writes Metrics| PG
    
    Mongo -->|Reads 'ingested' docs| Proxy
    Proxy -->|Writes Analytics| PG
    
    Docker -->|Logs| Promtail
    Promtail -->|Pushes Logs| Loki
    
    GitOpsAgent -->|Manages Code Synchronization| Hardware
    
    PG -->|Query Metrics| Grafana
    Loki -->|Query Logs| Grafana
```

## Detailed Architecture Documents

| Component | Description |
| :----------- | :------------- |
| **[Proxy Service](./proxy-service.md)** | Architecture of the Go-based API Gateway and ETL Engine. It bridges external data sources with the PostgreSQL. |
| **[System Metrics](./system-metrics.md)** | Details on the custom host telemetry collector (`gopsutil`). Pushes data directly to the `system_metrics` table in PostgreSQL (TimescaleDB). |
| **[Infrastructure](./infrastructure.md)** | Deployment (Docker), Storage (Postgres/Loki), and Security config. |
| **[GitOps Reconciliation](./../decisions/005-gitops-reconciliation-engine.md)** | Systemd-driven agent for automated, self-healing repository synchronization. |

## Related Documentation

- **[Decisions](../decisions/)**: Architecture Decision Records (ADRs).
- **[Planning](../planning/)**: Future trends and RFC drafts.
