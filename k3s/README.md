# k3s Cluster Manifests

This directory contains the Kubernetes manifests and Helm values for the Observability Hub cluster.

For detailed operational procedures, including deployment commands, image sideloading, and data migration, refer to the:

👉 **[k3s Operations Guide](../docs/notes/k3s-operations.md)**

## 📂 Directory Structure

- **collectors/**: Unified Host Telemetry Collector.
- **grafana/**: Visualization layer with persistence.
- **loki/**: Log storage and indexing.
- **minio/**: S3-compatible object storage for trace persistence.
- **opentelemetry/**: OpenTelemetry Collector for signal processing.
- **postgres/**: Relational data store (TimescaleDB/PostGIS).
- **prometheus/**: Time-series storage for infrastructure metrics.
- **tempo/**: Distributed trace storage.
- **thanos/**: Long-term Metrics Access.
- **namespace.yaml**: Core isolation boundary for the stack.
