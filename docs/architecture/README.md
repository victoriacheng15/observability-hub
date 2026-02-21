# Observability Hub Architecture

This directory contains the detailed architectural blueprints for the Observability Hub. The system follows a hybrid model, utilizing **Kubernetes (k3s)** for core data services and native Systemd units for host-level automation and data pipelines.

## 🗺️ System Context

The hub integrates standard observability tools with custom Go services to provide a resilient, self-healing telemetry platform orchestrated via Kubernetes.

```mermaid
graph TD
    subgraph External ["External Sources"]
        GH["GitHub Webhooks"]
        Mongo[(MongoDB Atlas)]
    end

    subgraph Host ["Host Environment"]
        CoreHostServices[Tailscale, Proxy, Custom Metrics, Secrets]
    end

    subgraph Cluster ["Data Platform (k3s)"]
        DataIngestion[Alloy & OpenTelemetry]
        ObservabilityStack[Loki, Tempo, Prometheus]
        Database[(PostgreSQL)]
        ObjectStore[(MinIO S3)]
    end

    subgraph Visualization ["User Interface"]
        Grafana[Grafana]
    end

    %% High-level Flow
    External --> CoreHostServices
    CoreHostServices --> DataIngestion
    CoreHostServices --> Database
    DataIngestion --> ObservabilityStack
    ObservabilityStack --> ObjectStore
    Cluster --> Visualization
```

---

## 📂 Documentation Domains

### 🧠 [Core Concepts](./core-concepts/)

Fundamental patterns and cross-cutting concerns that define how the system operates.

- **[Automation & GitOps](./core-concepts/automation.md)**: Webhook-driven reconciliation and self-healing patterns.
- **[Observability](./core-concepts/observability.md)**: Standards for JSON logging, Journald integration, and Alloy pipelines.

### 🏗️ [Infrastructure](./infrastructure/)

The runtime environment and foundational deployment strategies.

- **[Deployment Model](./infrastructure/deployment.md)**: Details on the hybrid Kubernetes/Systemd orchestration.
- **[Security](./infrastructure/security.md)**: Tailscale Funnel gating, HMAC authentication, and isolation boundaries.

### ⚙️ [Services](./services/)

Deep dives into the logic and implementation of specific system components.

- **[Proxy Service](./services/proxy.md)**: The API Gateway, Data Pipeline, and GitOps listener.
- **[Reading Sync](./services/reading-sync.md)**: The automated MongoDB to Postgres ETL pipeline.
- **[Second Brain](./services/second-brain.md)**: Knowledge ingestion from GitHub into PostgreSQL.
- **[System Metrics](./services/system-metrics.md)**: The host telemetry collector.
- **[Tailscale Gate](./services/tailscale-gate.md)**: Logic for the automated funnel gatekeeper.
