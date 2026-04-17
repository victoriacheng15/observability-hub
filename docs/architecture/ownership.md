# Platform Ownership Model

Observability Hub is organized as a closed-loop platform ownership system.

The operating model is:

```text
Source of Truth -> Runtime -> Signals -> Decisions -> Actions -> Memory
```

This page explains how the project connects infrastructure definition, deployment, observability, diagnosis, remediation, resource analysis, and institutional memory into one system.

## Ownership Loop

| Stage | Purpose | Project Surface |
| :--- | :--- | :--- |
| Source of Truth | Defines the intended platform state | `tofu/`, `k3s/`, `systemd/`, `.github/workflows/`, `Makefile` |
| Runtime | Runs the platform services | K3s workloads, host systemd services, databases, storage |
| Signals | Captures runtime behavior | OpenTelemetry, Prometheus, Loki, Tempo, Hubble, Grafana dashboards |
| Decisions | Turns signals into operator judgment | MCP tools, dashboards, runbooks, incident docs |
| Actions | Applies controlled remediation | GitOps sync, service restart, pod inspection, pod deletion, config patches |
| Memory | Preserves why and how the system changed | ADRs, RCAs, notes, workflows |

## Ownership Domains

| Domain | Source of Truth | Runtime | Signals | Diagnostic Path | Remediation Path | Memory |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Host Tier | `systemd/`, `scripts/`, `makefiles/systemd.mk` | `proxy`, OpenBao, host automation | systemd logs, host metrics | `hub_*` MCP tools, journal logs | service restart, unit update, script fix | `docs/notes/`, `docs/incidents/` |
| Cluster Tier | `k3s/`, `tofu/` | K3s workloads and namespaces | pod status, events, kube metrics | pod MCP tools, kube events | GitOps sync, rollout, pod deletion | `docs/workflows.md`, incident reports |
| Delivery | `.github/workflows/`, image tags, ArgoCD manifests | GitHub Actions, GHCR, ArgoCD | workflow status, image tags, sync state | workflow logs, proxy logs, ArgoCD state | PR fix, image retag, reconciliation | `docs/workflows.md` |
| Observability | `k3s/base/infra/`, telemetry config | OpenTelemetry, Loki, Tempo, Prometheus, Grafana | logs, metrics, traces, dashboards | telemetry MCP tools, Grafana queries | config patch, collector restart, datasource fix | observability docs, RCAs |
| Resource Efficiency | `k3s/`, `tofu/`, worker schedules | K3s workloads, host resources, storage systems | CPU, memory, disk, network, energy, workload metrics | Prometheus/Thanos queries, Grafana dashboards, worker analytics | resource limit patch, workload tuning, capacity plan | notes, RCAs, architecture docs |
| Data | `tofu/`, database manifests, backup config | Postgres, MinIO, object storage backups | DB health, PVC state, backup status | DB logs, dashboard panels, pod tools | failover, restore, storage fix | `docs/notes/postgres.md`, incidents |
| Networking | Cilium policies, network docs | Cilium, Hubble, service networking | flows, drops, DNS behavior | network MCP tools, flow baseline | policy patch, DNS/service correction | `docs/notes/network-flow-baseline.md` |
| Security | `config/openbao/`, secrets manifests | OpenBao, Kubernetes secrets, service accounts | auth failures, service logs, policy errors | host logs, pod logs, security docs | rotate secret, patch policy, tighten RBAC | security docs, ADRs |
| Agentic Ops | `cmd/mcp-obs-hub`, `internal/mcp/`, `skills/` | MCP tools over telemetry, pods, host, network | tool metrics, traces, logs | MCP tool calls and provider logs | bounded tool action, provider fix | MCP architecture docs, ADRs |
| Documentation | `docs/`, `AGENTS.md` | Versioned project memory | ADRs, RCAs, notes, workflow docs | doc index, linked incidents | update doc, add ADR/RCA | docs tree |

## Component Standard

Every platform component should be explainable with the same ownership questions:

- What is the component responsible for?
- Where is its desired state defined?
- How is it deployed or reconciled?
- What logs, metrics, traces, or flows prove it is healthy?
- What resource signals show capacity pressure, waste, or cost risk?
- Which tool or runbook diagnoses it?
- What is the safe remediation path?
- Where are decisions and incidents recorded?

This standard keeps the project from reading as a collection of tools. Each component has a place in the operating loop.
