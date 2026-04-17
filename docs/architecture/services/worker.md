# Unified Worker Architecture

The Unified Worker (`cmd/worker`, `internal/worker`) is a cluster-native execution engine designed for one-shot data processing tasks. It consolidates legacy analytics and ingestion services into a single, high-performance binary that executes specific domain logic based on a runtime mode flag.

This service represents the platform's batch-processing layer. It handles scheduled analytical and ingestion workloads without introducing multiple one-off services, which keeps the runtime model simpler and makes the telemetry story easier to follow.

For a quick mental model: the Worker is the system's scheduled execution engine for tasks that do not need to run continuously but still need strong observability, repeatability, and operational discipline. Its analytics mode turns infrastructure telemetry into capacity, efficiency, and cost-aware operating data.

## 🎯 Objective

To minimize operational overhead and resource fragmentation by providing a standardized, containerized environment for batch operations. The worker ensures that all non-continuous tasks follow the same lifecycle, telemetry, and security standards across the platform.

## 🧩 Execution Modes

The worker operates in two primary modes, triggered via the `--mode` CLI flag:

### 1. Analytics Mode (`--mode analytics`)

- **Mission**: Correlates infrastructure resource usage with platform outcomes so operators can reason about capacity, efficiency, and cost drivers from real telemetry.
- **Sources**:
  - **Thanos**: Retrieves energy (Kepler), Kubernetes, and host utilization metrics.
  - **Tailscale**: Inspects Funnel and mesh connectivity status.
- **Persistence**: Records high-fidelity resource samples into the PostgreSQL `analytics_metrics` table for trend analysis and operational reporting.
- **Scheduling**: Every 15 minutes via Kubernetes `CronJob`.

### 2. Ingestion Mode (`--mode ingestion`)

- **Mission**: Synchronizes external data sources into the Hub's local analytical store.
- **Sub-tasks**:
  - **Reading Sync**: Pulls article metadata and engagement metrics from MongoDB Atlas.
  - **Brain Sync**: Ingests journaling entries from GitHub Issues and calculates token metrics.
- **Persistence**: UPSERT operations in PostgreSQL relational tables.
- **Scheduling**: Daily at 02:00 via Kubernetes `CronJob`.

## ⚙️ Shared Logic & Lifecycle

- **Binary Structure**: Built as a thin entry point in `cmd/worker` that delegates to domain-isolated packages in `internal/worker/analytics` and `internal/worker/ingestion`.
- **Initialization**: Centralized discovery for environment variables, OpenBao secrets, and database connections (Postgres/Mongo).
- **Graceful Termination**: Implements a mandatory `telemetry.Shutdown()` defer loop to ensure that all metrics and trace batches are flushed before the short-lived Pod exits.

## 🔭 Observability Implementation

The worker is fully instrumented with the platform's OpenTelemetry Go SDK:

- **Dynamic Naming**: Service names are dynamically set to `worker.analytics` or `worker.ingestion` based on the execution mode.
- **Metric Catalog**: Emits `worker.batch.*` metrics (total, errors, duration) for operational tracking.
- **Distributed Tracing**: Root spans (`worker.run`) correlate internal processing steps with external API and database operations.
- **Structured Logging**: Emits JSON-formatted logs to Loki, including mode-specific metadata for easier filtering and alerting.

## 🛡️ Operational Excellence

- **GitOps Management**: Scheduled and reconciled via ArgoCD using standard Kubernetes `CronJob` manifests.
- **Idempotency**: All database operations (Postgres/Mongo) use `ON CONFLICT` or status-flag checks to ensure that accidental re-runs do not duplicate data or corrupt indices.
- **Resource Gating**: Isolated Pod resources (`requests/limits`) ensure that heavy ingestion tasks do not impact the stability of the cluster nodes.
