# k3s Cluster Manifests

This directory contains the Kubernetes manifests and Helm values for the Observability Hub cluster.

For detailed operational procedures, including deployment commands, image sideloading, and data migration, refer to the:

👉 **[k3s Operations Guide](../docs/notes/k3s-operations.md)**

## 📂 Directory Structure

- **alloy/**: Grafana Alloy DaemonSet for host-level telemetry.
- **grafana/**: Visualization layer with persistence.
- **loki/**: Log storage and indexing.
- **opentelemetry/**: OpenTelemetry Collector for signal processing.
- **postgres/**: Relational data store (TimescaleDB/PostGIS).
- **prometheus/**: Time-series storage for infrastructure metrics.
- **tempo/**: Distributed trace storage.
- **namespace.yaml**: Core isolation boundary for the stack.
