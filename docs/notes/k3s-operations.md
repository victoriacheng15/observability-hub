# k3s Operations Guide

This guide details the procedures for managing the observability stack within the k3s cluster, including deployment, image management, and data migration.

## 🚀 Component Management

### 1. Grafana Alloy (Telemetry Collector)

- **Manifest**: `k3s/alloy/manifest.yaml`
- **Values**: `k3s/alloy/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template alloy grafana/alloy -f k3s/alloy/values.yaml --namespace observability > k3s/alloy/manifest.yaml"
  kubectl apply -f k3s/alloy/manifest.yaml
  kubectl rollout restart daemonset alloy -n observability
  ```

### 2. Loki (Log Store)

- **Manifest**: `k3s/loki/manifest.yaml`
- **Values**: `k3s/loki/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template loki grafana/loki -f k3s/loki/values.yaml --namespace observability > k3s/loki/manifest.yaml"
  kubectl apply -f k3s/loki/manifest.yaml
  kubectl rollout restart statefulset loki -n observability
  ```

### 3. Grafana (Visualization)

- **Manifest**: `k3s/grafana/manifest.yaml`
- **Values**: `k3s/grafana/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template grafana grafana-community/grafana -f k3s/grafana/values.yaml --namespace observability > k3s/grafana/manifest.yaml"
  kubectl apply -f k3s/grafana/manifest.yaml
  kubectl rollout restart deployment grafana -n observability
  ```

### 4. PostgreSQL (Relational Data)

- **Manifest**: `k3s/postgres/manifest.yaml`
- **Values**: `k3s/postgres/values.yaml`
- **Update Command**:

  ```bash
  nix-shell --run "helm template postgres bitnami/postgresql -f k3s/postgres/values.yaml --namespace observability > k3s/postgres/manifest.yaml"
  kubectl apply -f k3s/postgres/manifest.yaml
  kubectl rollout restart statefulset postgres-postgresql -n observability
  ```

---

## 🖼️ Local Image Sideloading

Since we use a custom PostgreSQL image with extensions, it must be manually imported into the k3s node.

```bash
# 1. Build locally
docker build -t postgres-pod:latest -f docker/postgres/Dockerfile .

# 2. Export and Import
docker save -o postgres-pod.tar postgres-pod:latest
sudo k3s ctr images import postgres-pod.tar

# 3. Tag for consistency (removes docker.io/library prefix)
sudo k3s ctr images tag docker.io/library/postgres-pod:latest postgres-pod:latest

# 4. Cleanup
rm postgres-pod.tar
```

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
