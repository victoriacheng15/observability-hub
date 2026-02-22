# k3s Operations Guide

Reference: <https://github.com/grafana/helm-charts>

This guide details the procedures for managing the observability stack within the k3s cluster, including deployment, image management, and data migration. For automated updates (Regenerate -> Apply -> Restart), refer to the commands defined in `makefiles/k3s.mk`.

## 🚀 Component Management

### Collectors (Unified Host Telemetry)

- **Manifest**: `k3s/collectors/manifest.yaml`
- **Values**: `k3s/collectors/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template collectors k3s/collectors -f k3s/collectors/values.yaml --namespace observability > k3s/collectors/manifest.yaml"
  kubectl apply -f k3s/collectors/manifest.yaml
  kubectl rollout restart daemonset collectors -n observability
  ```

- **Local Image Sideloading**:
  Since this is a custom internal service, the image must be built and sideloaded into k3s.

```bash
# 1. Build locally
docker build -t collectors:v0.1.0 -f docker/collectors/Dockerfile .

# 2. Export and Import
docker save -o collectors.tar collectors:v0.1.0
sudo k3s ctr images import collectors.tar

# 3. Cleanup
rm collectors.tar
```

### Grafana (Visualization)

- **Manifest**: `k3s/grafana/manifest.yaml`
- **Values**: `k3s/grafana/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template grafana grafana-community/grafana -f k3s/grafana/values.yaml --namespace observability > k3s/grafana/manifest.yaml"
  kubectl apply -f k3s/grafana/manifest.yaml
  kubectl rollout restart deployment grafana -n observability
  ```

### Loki (Log Store)

- **Manifest**: `k3s/loki/manifest.yaml`
- **Values**: `k3s/loki/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template loki grafana/loki -f k3s/loki/values.yaml --namespace observability > k3s/loki/manifest.yaml"
  kubectl apply -f k3s/loki/manifest.yaml
  kubectl rollout restart statefulset loki -n observability
  ```

- **Notes**:
  - S3 credentials are injected from `minio-loki-secret` via environment variables
  - Loki Helm values use `${MINIO_LOKI_ACCESS_KEY}` and `${MINIO_LOKI_SECRET_KEY}` placeholders
  - Requires `-config.expand-env=true` flag (configured in `global.extraArgs`)
  - Secret is injected via `global.extraEnvFrom` in values.yaml

### OpenTelemetry (Collector)

- **Manifest**: `k3s/opentelemetry/manifest.yaml`
- **Values**: `k3s/opentelemetry/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template opentelemetry open-telemetry/opentelemetry-collector -f k3s/opentelemetry/values.yaml --namespace observability > k3s/opentelemetry/manifest.yaml"
  kubectl apply -f k3s/opentelemetry/manifest.yaml
  kubectl rollout restart deployment opentelemetry -n observability
  ```

### MinIO (S3 Storage Backend)

- **Manifest**: `k3s/minio/manifest.yaml`
- **Values**: `k3s/minio/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template minio minio/minio -f k3s/minio/values.yaml --namespace observability > k3s/minio/manifest.yaml"
  kubectl apply -f k3s/minio/manifest.yaml
  kubectl rollout restart deployment minio -n observability
  ```

### PostgreSQL (Relational Data)

- **Manifest**: `k3s/postgres/manifest.yaml`
- **Values**: `k3s/postgres/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template postgres bitnami/postgresql -f k3s/postgres/values.yaml --namespace observability > k3s/postgres/manifest.yaml"
  kubectl apply -f k3s/postgres/manifest.yaml
  kubectl rollout restart statefulset postgres-postgresql -n observability
  ```

- **Local Image Sideloading**:
  Since we use a custom PostgreSQL image with extensions, it must be manually imported into the k3s node.

```bash
# 1. Build locally
docker build -t postgres-pod:17.2.0-ext -f docker/postgres/Dockerfile .

# 2. Export and Import
docker save -o postgres-pod.tar postgres-pod:17.2.0-ext
sudo k3s ctr images import postgres-pod.tar

# 3. Tag for consistency
sudo k3s ctr images tag docker.io/library/postgres-pod:17.2.0-ext postgres-pod:17.2.0-ext

# 4. Cleanup
rm postgres-pod.tar
```

### Prometheus (Metrics Store)

- **Manifest**: `k3s/prometheus/manifest.yaml`
- **Values**: `k3s/prometheus/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template prometheus prometheus-community/prometheus -f k3s/prometheus/values.yaml --namespace observability > k3s/prometheus/manifest.yaml"
  kubectl apply -f k3s/prometheus/manifest.yaml
  kubectl rollout restart deployment prometheus-server -n observability
  ```

### Grafana Tempo (Trace Store)

- **Manifest**: `k3s/tempo/manifest.yaml`
- **Values**: `k3s/tempo/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template tempo grafana-community/tempo -f k3s/tempo/values.yaml --namespace observability > k3s/tempo/manifest.yaml"
  kubectl apply -f k3s/tempo/manifest.yaml
  kubectl rollout restart statefulset tempo -n observability
  ```

### Thanos Store Gateway (Long-term Metrics Storage)

- **Manifest**: `k3s/thanos/manifest.yaml`
- **Values**: `k3s/thanos/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template thanos bitnami/thanos -f k3s/thanos/values.yaml --namespace observability > k3s/thanos/manifest.yaml"
  kubectl apply -f k3s/thanos/manifest.yaml
  kubectl rollout restart statefulset thanos-storegateway -n observability
  ```

- **Notes**:
  - Helm-managed deployment using bitnami/thanos chart
  - Uses official quay.io/thanos/thanos:v0.32.2 image (not bitnami variant)
  - Requires existing secret: `minio-thanos-secret` (created via kubectl)
  - Secret contains S3 credentials for MinIO `prometheus-blocks` bucket
  - Store gateway only mode (querier, ruler, compactor, receive disabled)
  - Reference: [bitnami/thanos Helm Chart](https://github.com/bitnami/charts/tree/main/bitnami/thanos)

---

## 📊 Resource Limits Summary

| Component | CPU Req | RAM Req | CPU Limit | RAM Limit | Purpose |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **collectors** | 10m | 40Mi | 100m | 80Mi | Telemetry Collection |
| **grafana** | 10m | 64Mi | 100m | 128Mi | Visualization |
| **loki** | 100m | 256Mi | 300m | 640Mi | Log Storage |
| **minio** | 100m | 256Mi | 200m | 512Mi | S3 Storage Backend |
| **opentelemetry** | 20m | 100Mi | 150m | 256Mi | Trace Gateway |
| **postgres** | 250m | 512Mi | 500m | 768Mi | Relational Data |
| **prometheus** | 100m | 512Mi | 300m | 768Mi | Metrics Storage |
| **tempo** | 50m | 256Mi | 200m | 512Mi | Trace Storage |
| **thanos** | 100m | 256Mi | 200m | 512Mi | Long-term Metrics Access |

**Understanding Usage Totals:**

- **Mini Total (740m CPU / 2.12Gi RAM)**: The sum of all *Requests* (guaranteed resources).
- **Max Total (2.05 Cores / 4.18Gi RAM)**: The sum of all *Limits* (burst ceiling).

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
