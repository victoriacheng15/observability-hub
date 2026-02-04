# k3s Shadow Deployment

This directory contains Kubernetes manifests for the migration of the Observability Hub from Docker Compose to k3s, following the phased strategy defined in [ADR 011](../docs/decisions/011-phased-k3s-migration-strategy.md).

## 🗺️ Migration Roadmap

| Phase | Component | Status | Role |
| :--- | :--- | :--- | :--- |
| **Phase 1** | **Grafana Alloy** | 🟢 **Active** | **Telemetry Collector.** Scrapes logs and forwards to K3s-Loki. |
| **Phase 2** | **Loki** | 🟢 **Active** | **Log Store.** Replaced the Docker-Loki instance. |
| **Phase 3** | **Grafana** | ⚪ Planned | **Visualization.** "Pane of Glass" moved to the cluster. |
| **Phase 4** | **PostgreSQL** | 🟡 Shadowing | **Core Data.** Currently prototyping stateful persistence (`postgres-v2`). |

---

## 🚀 Phase 1: Grafana Alloy (Active)

We are using the "Strangler Fig" pattern: Alloy runs in k3s and sends data to the internal K3s-Loki instance.

### 1.1 Generate Alloy Manifests

We use Helm to template the complex DaemonSet configuration.

```bash
helm repo add grafana https://grafana.github.io/helm-charts
```

**Note:** Connects your local environment to the official Grafana repository so you can download the latest Alloy charts.

```bash
helm template alloy grafana/alloy \
  -f k3s/alloy/values.yaml \
  --namespace observability \
  > k3s/alloy/manifest.yaml
```

**Note:** This command processes the official chart using our custom settings:

- `-f values.yaml`: Configures the infrastructure (DaemonSet, Volume Mounts).
- `--set-file ...`: Injects your custom River (HCL) scraping logic into the manifest.
- `> manifest.yaml`: Saves the output to a file so it can be versioned and inspected before applying.

### 1.2 Deploy Alloy

```bash
kubectl apply -f k3s/namespace.yaml
```

**Note:** Creates the `observability` namespace to isolate our monitoring tools from other cluster workloads.

```bash
kubectl apply -f k3s/alloy/manifest.yaml
```

**Note:** Applies the generated manifest to the cluster, spinning up the Alloy pods on every node.

### 1.3 Verify Alloy

```bash
kubectl logs -l app.kubernetes.io/name=alloy -n observability -f
```

**Note:** Streams logs from the Alloy pods. Look for "Alloy is running" to confirm the telemetry engine is active.

### 1.4 Reload Configuration

If you update `k3s/alloy/values.yaml` and re-apply the manifest, you must restart the pods to pick up the new ConfigMap:

```bash
kubectl rollout restart daemonset alloy -n observability
```

---

## 🏗️ Phase 2: Loki (Active)

Log storage is now running inside the cluster with persistent storage.

### 2.1 Generate Loki Manifests

```bash
helm template loki grafana/loki \
  --namespace observability \
  -f k3s/loki/values.yaml \
  > k3s/loki/manifest.yaml
```

### 2.2 Deploy Loki

```bash
kubectl apply -f k3s/loki/manifest.yaml
```

### 2.3 Verify Persistence

```bash
kubectl get pvc -n observability
```

**Note:** Confirm the `storage-loki-0` volume is `Bound`.

### 2.4 Migration: Docker to K3s Data Transfer

If migrating from Docker Compose, use these commands to copy the Loki data volume to the K3s PersistentVolume.

```bash
# 1. Identify Paths
DOCKER_PATH=$(docker volume inspect loki_data --format '{{.Mountpoint}}')
K3S_PATH=$(kubectl get pv $(kubectl get pvc storage-loki-0 -n observability -o jsonpath='{.spec.volumeName}') -o jsonpath='{.spec.local.path}')

# 2. Scale Down (Stop Writes)
kubectl scale statefulset loki -n observability --replicas=0

# 3. Copy Data (Archive Mode)
sudo cp -a "$DOCKER_PATH/." "$K3S_PATH/"

# 4. Fix Permissions (UID 10001 = Loki User)
sudo chown -R 10001:10001 "$K3S_PATH"

# 5. Scale Up
kubectl scale statefulset loki -n observability --replicas=1
```

---

## 📊 Phase 3: Grafana (Planned)

In this phase, we move the "Pane of Glass" into the cluster.

### 3.1 Generate Grafana Manifests

```bash
helm template grafana grafana/grafana \
  --namespace observability \
  -f k3s/grafana/values.yaml \
  > k3s/grafana/manifest.yaml
```

**Key Configuration (`values.yaml`):**

- **Persistence:** Enabled (10Gi) to save dashboards/users.
- **Datasources:** Automatically provisions `Loki` (URL: `http://loki-gateway.observability:80`).
- **Service:** `NodePort` for external access (e.g., port 30000).

### 3.2 Deploy Grafana

```bash
kubectl apply -f k3s/grafana/manifest.yaml
```

### 3.3 Access UI

1. **Get Admin Password:**

   ```bash
   kubectl get secret --namespace observability grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
   ```

2. **Open Browser:**
   Navigate to `http://<SERVER_IP>:30000` (or configured NodePort).

---

## 🧪 Phase 4: PostgreSQL (Shadow Prototype)

*Note: This is strictly for testing storage patterns. It does not yet hold production data.*

### 4.1 Import Image

```bash
docker save -o postgres_pod.tar postgres_pod:latest
```

**Note:** Saves your local Docker build into a tarball archive.

```bash
sudo k3s ctr images import postgres_pod.tar
```

**Note:** Sideloads the tarball into the internal k3s registry so the cluster can run the image without an external pull.

```bash
rm postgres_pod.tar
```

**Note:** Removes the temporary archive to save disk space.

### 4.2 Deploy Prototype

```bash
kubectl apply -f k3s/postgres/manifest.yaml
```

**Note:** Deploys the experimental database resources.

---

## 🛠️ General Cluster Commands

### 1. Check Resources

```bash
kubectl get pods -A
# or 
kubectl get pods -n observability -o wide
```

**Note:** Lists all pods in the cluster across all namespaces.

```bash
kubectl get svc -A
```

**Note:** Lists all services in the cluster, showing their internal and external (NodePort) IP addresses.

### 2. Monitor State

```bash
nix-shell --run "k9s"
```

**Note:** Launches the terminal UI for real-time cluster management.
