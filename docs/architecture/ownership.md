# Platform Ownership Model

Observability Hub is organized around a closed-loop ownership model:

```text
Source of Truth -> Runtime -> Signals -> Decisions -> Actions -> Memory
```

The purpose of this page is routing. It should help an operator decide where a responsibility lives, how to diagnose it, and where a durable change should be made.

## Ownership Loop

| Stage | Purpose | Project Surface |
| :--- | :--- | :--- |
| Source of Truth | Defines intended state | `tofu/`, `k3s/`, `systemd/`, `.github/workflows/`, `Makefile` |
| Runtime | Runs the platform | K3s workloads, host services, databases, retained storage |
| Signals | Captures behavior | OpenTelemetry, Prometheus, Loki, Tempo, Hubble, Grafana |
| Decisions | Turns signals into judgment | MCP tools, dashboards, runbooks, incidents |
| Actions | Applies controlled remediation | GitOps sync, Tofu apply, service restart, pod operation, config change |
| Memory | Preserves why the system changed | ADRs, RCAs, notes, workflows |

## Control Plane

The control plane owns how changes enter and reconcile through the system.

| Area | Owns | Diagnose With | Change Through |
| :--- | :--- | :--- | :--- |
| GitOps | ArgoCD apps, sync state, cluster manifest reconciliation | ArgoCD UI, app status, proxy logs | `k3s/`, GitHub PRs |
| Infrastructure | Tofu-managed Helm releases, namespaces, storage classes, cloud-backed state | `tofu plan`, pod status, service status | `tofu/` |
| Delivery | GitHub Actions, GHCR images, image tags, deployment references | workflow logs, image tags, ArgoCD sync | `.github/`, `docker/`, `k3s/base/*/kustomization.yaml` |
| Host Services | Proxy, OpenBao, host automation, local service units | journal logs, service status, host metrics | `systemd/`, `scripts/`, `makefiles/` |

## Data Plane

The data plane owns runtime state, telemetry, and the durable stores the platform depends on.

| Area | Owns | Diagnose With | Change Through |
| :--- | :--- | :--- | :--- |
| Database | CNPG Postgres, database services, Azure-backed backups | pod tools, DB logs, backup status | CNPG manifests, `tofu/` |
| Telemetry | OpenTelemetry, Prometheus, Loki, Tempo, retained local PVCs | MCP telemetry tools, Grafana, pod logs | `k3s/base/infra/`, `tofu/` |
| Visualization | Grafana dashboards, datasources, operational views | Grafana UI, datasource checks, pod logs | Grafana values, dashboard files |
| Resource Analytics | Worker analytics, host and cluster resource metrics, capacity history | Prometheus queries, worker logs, analytics tables | `internal/worker/`, `k3s/base/worker/` |

## Operations Plane

The operations plane owns diagnosis, response, safety boundaries, and institutional memory.

| Area | Owns | Diagnose With | Change Through |
| :--- | :--- | :--- | :--- |
| Network | Cilium policies, Hubble flows, service connectivity | Hubble, MCP network tools, flow baseline | `k3s/cilium-policies/`, network docs |
| Agentic Ops | MCP tools, providers, capability reporting | MCP tool calls, gateway logs, provider health | `cmd/mcp-obs-hub/`, `internal/mcp/` |
| Security | OpenBao config, service accounts, secrets posture, access controls | host logs, pod logs, security docs | `config/openbao/`, Kubernetes manifests |
| Memory | ADRs, RCAs, plans, runbooks, workflow docs | docs search, linked incidents, commit history | `docs/`, `AGENTS.md` |

## Operating Rules

- Source-of-truth changes should go through Git, Tofu, or ArgoCD depending on the ownership plane.
- Runtime fixes should be followed by a source-of-truth update when the fix changes intended state.
- Incidents that reveal architecture or operating-model gaps should update ADRs, RCAs, or notes.
- Ownership docs should stay high level; implementation details belong in service, workflow, or incident docs.
