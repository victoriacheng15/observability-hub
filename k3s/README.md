# k3s Shadow Deployment

This directory contains Kubernetes manifests for the "v2" shadow deployment of the Observability Hub services, as defined in [ADR 007](../docs/decisions/007-k3s-shadow-deployment-orchestration.md).

## Services

### 1. PostgreSQL (v2)

- **Location:** `k3s/postgres/`
- **Image:** `postgres_pod:latest`
- **Service Type:** NodePort (`30432`)

### 2. Import into k3s

```bash
docker save -o postgres_pod.tar postgres_pod:latest
sudo k3s ctr images import postgres_pod.tar
rm postgres_pod.tar
```

### 3. Apply Manifests

Deploy the services to the k3s cluster.

```bash
kubectl apply -f k3s/postgres/manifest.yaml
```

## Verification

Verify the pods are running and services are accessible via their NodePorts.

```bash
kubectl get pods
kubectl get svc
```
