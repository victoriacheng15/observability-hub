# Observability Hub Architecture

This directory contains the detailed architectural blueprints for the Observability Hub. The system follows a hybrid model, utilizing Docker for core data services and native Systemd units for host-level automation and data pipelines.

## 🗺️ System Context

The hub integrates standard observability tools with custom Go services to provide a resilient, self-healing telemetry platform.

```mermaid
graph TD
    subgraph "External Sources"
        GitHub[GitHub Webhooks]
        Mongo[(MongoDB Atlas)]
    end

    subgraph HostEnvironment [Host Environment]
        Hardware[Host Hardware]
        subgraph HostServices [Native Services]
            Proxy[Proxy Service]
            Gate[Tailscale Gate]
            Metrics[Metrics Collector]
            Bao[OpenBao Secret Store]
        end
    end

    subgraph DataPlatform [Data Platform (Docker)]
        direction TB
        Promtail[Promtail]
        Loki[(Loki)]
        PG[(PostgreSQL)]
        Grafana[Grafana]
    end

    %% Data Flow
    Bao -.->|Secrets| HostServices
    GitHub -->|Webhooks| Proxy
    Proxy -->|Executes| Sync[gitops_sync.sh]
    Mongo -->|Data| Proxy
    Proxy -->|Writes| PG
    Hardware -->|Telemetry| Metrics
    Metrics -->|Writes| PG

    %% Logging
    HostServices -.->|Journal| Promtail
    Promtail -->|Pushes| Loki
    PG -->|Visualizes| Grafana
    Loki -->|Visualizes| Grafana
```

---

## 📂 Documentation Domains

### 🧠 [Core Concepts](./core-concepts/)

Fundamental patterns and cross-cutting concerns that define how the system operates.

- **[Automation & GitOps](./core-concepts/automation.md)**: Webhook-driven reconciliation and self-healing patterns.
- **[Observability](./core-concepts/observability.md)**: Standards for JSON logging, Journald integration, and Promtail pipelines.

### 🏗️ [Infrastructure](./infrastructure/)

The runtime environment and foundational deployment strategies.

- **[Deployment Model](./infrastructure/deployment.md)**: Details on the hybrid Docker/Systemd orchestration.
- **[Security](./infrastructure/security.md)**: Tailscale Funnel gating, HMAC authentication, and isolation boundaries.

### ⚙️ [Services](./services/)

Deep dives into the logic and implementation of specific system components.

- **[Proxy Service](./services/proxy.md)**: The API Gateway, Data Pipeline, and GitOps listener.
- **[System Metrics](./services/system-metrics.md)**: The host telemetry collector.
- **[Tailscale Gate](./services/tailscale-gate.md)**: Logic for the automated funnel gatekeeper.
