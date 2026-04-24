# k3s Cluster Manifests

This directory contains the Kubernetes manifests and Helm values for the Observability Hub cluster.

For detailed operational procedures, including deployment commands, image sideloading, and data migration, refer to the:

👉 **[k3s Operations Guide](../docs/notes/k3s-operations.md)**

## 📂 Directory Structure

- **base/**: Shared Kustomize base for cluster workloads and infrastructure.
- **base/hardware-sim/**: Synthetic hardware simulation workloads such as
  `sensor-fleet`, `chaos-controller`, and `mqtt-ingestor`.
- **base/hub-apps/**: Hub application workloads managed in-cluster.
- **base/infra/**: Helm values and provisioned assets for Grafana, Loki, MinIO,
  OpenTelemetry, Prometheus, Tempo, and Thanos.
- **base/rbac/**: Shared service accounts, roles, and bindings.
- **base/worker/**: Worker CronJobs and their base image/tag configuration.
- **bootstrap/**: Cluster bootstrap assets.
- **cilium-policies/**: Network policy definitions for Cilium.
- **overlays/dev/**: Development overlay for namespace, image, and rollout
  overrides.
- **overlays/prod/**: Production overlay for namespace, image, and rollout
  overrides.
