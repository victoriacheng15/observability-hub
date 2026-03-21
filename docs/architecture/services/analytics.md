# Analytics Engine Architecture

The Analytics Engine (`cmd/analytics`, `internal/analytics`) is a unified **Resource-to-Value and Efficiency Service** deployed as a Kubernetes DaemonSet. Its primary mission is to correlate infrastructure resource consumption (Energy, Cost) with platform outcomes (Syncs, Tasks) to calculate holistic efficiency and environmental impact.

## 🎯 Objective

To establish a high-fidelity "Resource-to-Value" framework that enables unit economics auditing and automated Carbon Debt tracking. The engine transforms raw infrastructure signals into actionable business telemetry.

## 🧩 Component Details

- **Type**: Kubernetes DaemonSet.
- **Sources**:
  - **Thanos**: Retrieves Kepler energy metrics and host utilization data.
  - **Tailscale**: Gathers network status and Funnel connectivity data.
- **Destinations**:
  - **PostgreSQL**: Persists structured resource analytics to the `analytics_metrics` table for relational correlation.
  - **OpenTelemetry Collector**: Forwards service-level traces and internal operational metrics.
- **Deployment**: Fully managed via **OpenTofu** as a native Kubernetes `DaemonSet` resource, utilizing a side-loaded container image for host-level telemetry access.

## ⚙️ Logic Flow

1. **Host Context Detection**: Identifies the underlying OS and hardware environment (e.g., AMD Ryzen vs. Intel) to optimize energy queries.
2. **Resource Ingestion**:
    - Queries Thanos for the 15-minute increase in `kepler_node_cpu_joules_total`.
    - Retrieves utilization baselines for CPU, RAM, and Disk.
3. **Value Correlation Loop**:
    - **Financial**: Converts Joules to CAD based on dynamic cost factors.
    - **Environmental**: Calculates Carbon Debt (gCO2) using regional grid intensity factors.
    - **Efficiency**: (Phase 4) Joins resource data with successful operation counts from the Proxy/Ingestion databases.
4. **Persistence**: Records high-fidelity samples into the relational `analytics_metrics` table, using a specialized `metric_kind` schema for fast analytical joins.

## 🔭 Observability Implementation

The Analytics Engine is the "Ground Truth" provider for the platform's sustainability metrics.

- **Metric Namespace**: Uses the `analytics.*` root (e.g., `analytics.collection.total`, `analytics.tailscale.active`).
- **Relational Analytics**: Exposes structured data via PostgreSQL, enabling AI agents (MCP) to perform deep "Eco-Audits" and efficiency predictions.
- **Unit Economics**: Directly calculates "Joules-per-Sync" and "Cost-per-Thought" to justify infrastructure spend and observability overhead.
