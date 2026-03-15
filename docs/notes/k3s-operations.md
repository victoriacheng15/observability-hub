# k3s Operations Guide

Reference: <https://github.com/grafana/helm-charts>

This guide details the procedures for managing the observability stack within the k3s cluster, including deployment, image management, and data migration. For automated updates (Regenerate -> Apply -> Restart), refer to the commands defined in `makefiles/k3s.mk`.

## 🚀 Component Management

### 🔄 Update & Maintenance Workflow

To maintain the observability stack, follow this three-step lifecycle for all Tofu-managed components.

#### Step 1: Check Current vs. Latest Versions

Verify what is currently running and what is available in the repositories.

```bash
# 1.1 View CURRENTly installed versions
helm list -n observability
# CHART column: name-version (e.g., grafana-10.5.15)
# APP VERSION column: software version (e.g., 12.3.1)

# 1.2 Update Helm repository cache
helm repo update

# 1.3 View LATEST available versions
helm search repo grafana-community/grafana && \
helm search repo grafana/loki && \
helm search repo open-telemetry/opentelemetry-collector && \
helm search repo minio/minio && \
helm search repo cloudnative-pg/cloudnative-pg && \
helm search repo prometheus-community/prometheus && \
helm search repo grafana-community/tempo && \
helm search repo bitnami/thanos
```

#### Step 2: Update Configuration

If an update is available, manually update the corresponding file:

- **Tofu**: Edit versions in `tofu/storage.tf`, `tofu/databases.tf`, `tofu/metrics.tf`, `tofu/telemetry.tf`, or `tofu/hub.tf`.
- **Values**: If necessary, update configurations in `k3s/<component>/values.yaml`.

#### Step 3: Plan & Apply Changes

Apply the new configuration to the cluster.

```bash
tofu plan
tofu apply
```

---

### Analytics (Unified Host Telemetry)

- **Tofu Configuration**: `tofu/hub.tf`
- **Notes**:
  - Managed as a native `kubernetes_daemon_set_v1` resource.
  - Requires local image building and sideloading (see below).

- **Local Image Sideloading**:

```bash
# 1. Build locally (using podman)
podman build -t analytics:v0.1.0 -f docker/analytics/Dockerfile .

# 2. Export and Import
podman save -o analytics.tar localhost/analytics:v0.1.0
sudo k3s ctr images import analytics.tar

# 3. Tag for K3s local lookup
sudo k3s ctr images tag localhost/analytics:v0.1.0 analytics:v0.1.0
sudo k3s ctr images tag localhost/analytics:v0.1.0 docker.io/library/analytics:v0.1.0

# 4. Cleanup
rm analytics.tar
```

### Grafana (Visualization)

- **Chart**: `grafana-community/grafana`
- **Tofu Configuration**: `tofu/hub.tf`
- **Values**: `k3s/grafana/values.yaml`

### Loki (Log Store)

- **Chart**: `grafana/loki`
- **Tofu Configuration**: `tofu/telemetry.tf`
- **Values**: `k3s/loki/values.yaml`
- **Notes**:
  - S3 credentials are injected from `minio-loki-secret` via environment variables
  - Loki Helm values use `${MINIO_LOKI_ACCESS_KEY}` and `${MINIO_LOKI_SECRET_KEY}` placeholders
  - Requires `-config.expand-env=true` flag (configured in `global.extraArgs`)
  - Secret is injected via `global.extraEnvFrom` in values.yaml

### OpenTelemetry (Collector)

- **Chart**: `open-telemetry/opentelemetry-collector`
- **Tofu Configuration**: `tofu/telemetry.tf`
- **Values**: `k3s/opentelemetry/values.yaml`

### MinIO (S3 Storage Backend)

- **Chart**: `minio/minio`
- **Tofu Configuration**: `tofu/databases.tf`
- **Values**: `k3s/minio/values.yaml`

### HA PostgreSQL (CloudNativePG)

- **Chart**: `cloudnative-pg/cloudnative-pg`
- **Tofu Configuration**: `tofu/databases.tf`
- **Backup Destination**: Azure Blob Storage (`pg-backup` container)
- **Local Image Sideloading**:

```bash
# 1. Build locally
podman build -t postgres-cnpg:17 -f docker/postgres-cnpg/Dockerfile .

# 2. Export and Import
podman save -o postgres-cnpg.tar localhost/postgres-cnpg:17
sudo k3s ctr images import postgres-cnpg.tar

# 3. Tag for consistency
sudo k3s ctr images tag localhost/postgres-cnpg:17 postgres-cnpg:17
sudo k3s ctr images tag localhost/postgres-cnpg:17 docker.io/library/postgres-cnpg:17

# 4. Cleanup
rm postgres-cnpg.tar
```

### Prometheus (Metrics Store)

- **Chart**: `prometheus-community/prometheus`
- **Tofu Configuration**: `tofu/metrics.tf`
- **Values**: `k3s/prometheus/values.yaml`

### Grafana Tempo (Trace Store)

- **Chart**: `grafana-community/tempo`
- **Tofu Configuration**: `tofu/telemetry.tf`
- **Values**: `k3s/tempo/values.yaml`

### Thanos Store Gateway (Long-term Metrics Storage)

- **Chart**: `bitnami/thanos`
- **Tofu Configuration**: `tofu/metrics.tf`
- **Values**: `k3s/thanos/values.yaml`
- **Notes**:
  - Uses official quay.io/thanos/thanos:v0.32.2 image (not bitnami variant)
  - Requires existing secret: `minio-thanos-secret` (created via kubectl)
  - Secret contains S3 credentials for MinIO `prometheus-blocks` bucket
  - Store gateway only mode (querier, ruler, compactor, receive disabled)
  - Reference: [bitnami/thanos Helm Chart](https://github.com/bitnami/charts/tree/main/bitnami/thanos)

## 🔌 Connectivity Bridge (MCP Era)

The platform utilizes **NodePort** to bridge host-based services (MCP agents, proxy, ingestion) with the K3s cluster via `localhost`.

| Service | Protocol | NodePort | Target URI (Host-View) |
| :--- | :--- | :--- | :--- |
| **Grafana** | HTTP | 30000 | `http://localhost:30000` |
| **Loki (Gateway)** | HTTP | 30100 | `http://localhost:30100` |
| **Thanos (Query)** | HTTP | 30090 | `http://localhost:30090` |
| **Tempo** | HTTP | 30200 | `http://localhost:30200` |
| **OTel Collector**| gRPC | 30317 | `localhost:30317` |
| **PostgreSQL** | TCP | 30432 | `localhost:30432` |

---

## 📊 Resource Limits Summary

- *Last Updated: 2026-03-15 (Standardized via _standards.yaml)*

| Component | Profile | CPU Req | RAM Req | CPU Limit | RAM Limit | Purpose |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **analytics** | Custom | 100m | 200Mi | 100m | 400Mi | Telemetry Collection |
| **grafana** | Medium | 50m | 256Mi | 200m | 512Mi | Visualization |
| **loki** | Large | 200m | 512Mi | 1000m | 2Gi | Log Storage |
| **minio** | Large | 200m | 512Mi | 1000m | 2Gi | S3 Storage Backend |
| **opentelemetry** | Medium | 50m | 256Mi | 200m | 512Mi | Trace Gateway |
| **postgres** | Custom | 100m | 512Mi | 500m | 1Gi | Relational Data |
| **prometheus** | Large | 200m | 512Mi | 1000m | 2Gi | Metrics Storage |
| **tempo** | Large | 200m | 512Mi | 1000m | 2Gi | Trace Storage |
| **thanos** | Medium | 50m | 256Mi | 200m | 512Mi | Long-term Metrics Access |

**Understanding Usage Totals:**

- **Mini Total (~1.16 Cores / 3.75Gi RAM)**: The sum of all *Requests* (guaranteed resources).
- **Max Total (~5.25 Cores / 11.25Gi RAM)**: The sum of all *Limits* (burst ceiling).

---

## 💾 Data Migration (Docker to k3s)

Procedure for migrating persistent volumes from standalone Docker to k3s PVCs.

### Standard Volume Sync Pattern

1. **Identify Paths**:
    - `DOCKER_PATH`: `$(docker volume inspect <vol_name> --format '{{.Mountpoint}}')`
    - `K3S_PATH`: `$(kubectl get pv $(kubectl get pvc <pvc_name> -n observability -o jsonpath='{.spec.volumeName}') -o jsonpath='{.spec.local.path}')`
2. **Stop Writes**: Scale the k3s resource to 0 and stop the Docker container.
3. **Sync Data**: `sudo cp -a "$DOCKER_PATH/." "$K3S_PATH/"`
    - *Note for PostgreSQL*: Copy into the `$K3S_PATH/data` subdirectory.
4. **Fix Permissions**: `sudo chown -R <uid>:<gid> "$K3S_PATH"`
    - Loki: `10001:10001`
    - Grafana: `472:472`
    - PostgreSQL: `999:999`
5. **Scale Up**: Scale the k3s resource back to its original replica count.

---

## 🛠️ General Troubleshooting

- **Check Pods**: `kubectl get pods -n observability`
- **Check Logs**: `kubectl logs <pod_name> -n observability`
- **Cluster TUI**: `nix-shell --run "k9s"`
