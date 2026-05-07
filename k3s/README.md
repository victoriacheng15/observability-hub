# k3s Cluster Manifests

This directory contains the Kubernetes manifests and Helm values for the Observability Hub cluster.

For detailed operational procedures, including deployment commands, image sideloading, and data migration, refer to the:

👉 **[k3s Operations Guide](../docs/notes/k3s-operations.md)**

## 📂 Directory Structure

- **base/**: Shared Kustomize base for cluster workloads and infrastructure.
- **base/hub-apps/**: Hub-managed application objects, including Argo CD child
  apps that point at external workload repos such as `hardware-sim-lab`.
- **base/infra/**: Helm values and provisioned assets for Grafana, Loki, MinIO,
  OpenTelemetry, Prometheus, and Tempo.
- **base/rbac/**: Shared platform service accounts, roles, and bindings.
- **base/worker/**: Worker CronJobs and their base image/tag configuration.
- **bootstrap/**: Cluster bootstrap assets.
- **cilium-policies/**: Network policy definitions for Cilium.

Hardware simulation workloads no longer live directly in this tree. They are
managed from the public `hardware-sim-lab` repo through Argo CD child
applications declared under `base/hub-apps/`.
