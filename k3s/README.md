# k3s Shadow Deployment

This directory contains Kubernetes manifests for the migration of the Observability Hub from Docker Compose to k3s, following the phased strategy defined in [ADR 011](../docs/decisions/011-phased-k3s-migration-strategy.md).

## 🗺️ Migration Roadmap

| Phase | Component | Status | Role |
| :--- | :--- | :--- | :--- |
| **Phase 1** | **Grafana Alloy** | 🟢 **Active** | **Telemetry Collector.** Scrapes logs and forwards to Docker-Loki. |
| **Phase 2** | **Loki** | ⚪ Planned | **Log Store.** Will replace the Docker-Loki instance. |
| **Phase 3** | **Grafana** | ⚪ Planned | **Visualization.** "Pane of Glass" moved to the cluster. |
| **Phase 4** | **PostgreSQL** | 🟡 Shadowing | **Core Data.** Currently prototyping stateful persistence (`postgres-v2`). |

---

## 🚀 Phase 1: Grafana Alloy (Active)

We are using the "Strangler Fig" pattern: Alloy runs in k3s but sends data to the existing Docker-Loki instance.

### Generate Manifests

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

### 2. Deploy

```bash
kubectl apply -f k3s/namespace.yaml
```

**Note:** Creates the `observability` namespace to isolate our monitoring tools from other cluster workloads.

```bash
kubectl apply -f k3s/alloy/manifest.yaml
```

**Note:** Applies the generated manifest to the cluster, spinning up the Alloy pods on every node.

### 3. Verify

```bash
kubectl logs -l app.kubernetes.io/name=alloy -n observability -f
```

**Note:** Streams logs from the Alloy pods. Look for "Alloy is running" to confirm the telemetry engine is active.

---

## 🏗️ Phase 2: Loki (Planned)

In this phase, we move the log storage into the cluster. This involves setting up persistent storage and updating Alloy to use internal cluster networking.

### Generate Manifests (Loki)

We will use the official Grafana Loki chart. Persistence will be handled via a `PersistentVolumeClaim` (PVC) using the local-path provisioner.

```bash
helm template loki grafana/loki \
  --namespace observability \
  -f k3s/loki/values.yaml \
  > k3s/loki/manifest.yaml
```

### Configure Persistence

Ensure the `values.yaml` defines a standard single-binary deployment with filesystem storage:

- `loki.storage.type: filesystem`
- `loki.persistence.enabled: true`
- `loki.persistence.size: 10Gi`

### 3. Update Alloy

Once Loki is running, update `k3s/alloy/values.yaml` to point to the internal service:

```river
loki.write "local_loki" {
  endpoint {
    url = "http://loki.observability.svc.cluster.local:3100/loki/api/v1/push"
  }
}
```

**Note:** We can then disable `hostNetwork: true` in Alloy as it no longer needs to reach the host IP.

### 4. Verify

```bash
kubectl get pvc -n observability
```

**Note:** Confirm the volume is bound. Then check Loki logs to ensure it is accepting writes from Alloy.

---

## 📊 Phase 3: Grafana (Planned)

*Placeholder: Manifests will be added once the data layer (Loki/Postgres) is stable.*

---

## 🧪 Phase 4: PostgreSQL (Shadow Prototype)

*Note: This is strictly for testing storage patterns. It does not yet hold production data.*

### 1. Import Image

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

### 2. Deploy Prototype

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
