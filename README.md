# Observability Hub

## What is this?

Observability Hub is a closed-loop platform ownership system for a self-hosted Kubernetes environment.

It is built to show end-to-end ownership, not just tool installation: infrastructure definition, deployment automation, runtime observability, incident diagnosis, safe remediation, and operational memory.

Git and infrastructure definitions describe the intended state. Host and cluster runtimes execute that state. Telemetry systems expose behavior. MCP tools and dashboards support diagnosis. Remediation flows apply controlled fixes. ADRs, RCAs, and notes preserve what was learned.

It demonstrates how a platform engineering team owns a system end to end:

- provision infrastructure declaratively
- deploy services through GitOps
- collect logs, metrics, traces, and network signals
- diagnose failures with dashboards, runbooks, and MCP tools
- remediate safely through bounded operational paths
- preserve decisions and incidents as operational memory

The core loop is:

```mermaid
flowchart TB
    Source["Source of Truth<br/>Git, OpenTofu, Kustomize, systemd"]
    Runtime["Runtime<br/>K3s, host services, databases"]
    Signals["Signals<br/>OTel, Prometheus, Loki, Tempo, Hubble"]
    Decisions["Decisions<br/>Grafana, MCP tools, runbooks"]
    Actions["Actions<br/>GitOps sync, pod repair, service restart"]
    Memory["Memory<br/>ADRs, RCAs, notes, workflows"]

    Source --> Runtime
    Runtime --> Signals
    Signals --> Decisions
    Decisions --> Actions
    Actions --> Source
    Decisions --> Memory
    Memory --> Source
```

The goal is to make the ownership loop visible first, so the project reads as a complete platform engineering system rather than a collection of DevOps tools.

- 🌐 [Project Portal](https://victoriacheng15.github.io/observability-hub/)  

---

## 🔍 What I Built (Quick Proof)

- Kubernetes (K3s) homelab running 10+ platform components
- GitOps deployment using Argo CD (App-of-Apps pattern)
- Full observability: logs, metrics, traces (OpenTelemetry + Grafana stack)
- Agent-readable operations through MCP tools for telemetry, pods, host health, and network flows
- High-availability PostgreSQL with automated failover (CloudNativePG)
- Centralized dashboards for monitoring and debugging
- Secrets management without hardcoding credentials
- Infrastructure as Code using OpenTofu (layered architecture)
- Data ingestion pipeline with worker-based processing
- eBPF-based networking and visibility using Cilium
- Backup and storage integration with Azure Blob Storage + MinIO

---

## 📦 Platform Projects

This platform is built as connected ownership domains:

| Domain | What It Proves |
| :--- | :--- |
| GitOps Deployment | Declarative cluster management with Argo CD self-healing |
| Observability Stack | Prometheus, Grafana, Loki, Tempo dashboards and alerts |
| Telemetry Pipeline | OpenTelemetry logs, metrics, and traces across services |
| High Availability Database | PostgreSQL failover with Azure Blob Storage backups |
| Secrets Management | Dynamic secrets and policy management with OpenBao |
| Infrastructure as Code | Layered OpenTofu architecture for infrastructure ownership |
| Networking | Cilium eBPF visibility, policy control, and flow debugging |
| CI/CD | GitHub Actions, image publication, and GitOps reconciliation |
| Incident Response | Diagnostics, bounded repair actions, RCAs, and runbooks |
| Data Ingestion | Worker-based batch processing and analytics jobs |

---

## 🧠 Problems I Solved

| Problem | Solution |
| :--- | :--- |
| Manual deployments | GitOps automation with Argo CD and webhook-triggered reconciliation |
| No visibility into systems | Logs, metrics, traces, network flows, and Grafana dashboards |
| Secrets stored in code | Dynamic secret management with OpenBao |
| Single point of failure | High-availability PostgreSQL and backup paths |
| Hard-to-debug issues | MCP diagnostics, dashboards, runbooks, and incident reports |
| Infrastructure drift | Declarative source of truth with OpenTofu, Kustomize, and GitOps |
| Operational knowledge loss | Versioned ADRs, RCAs, notes, and workflow docs |

---

## Documentation Map

| Area | Purpose |
| :--- | :--- |
| [Full Documentation](./docs/README.md) | Central docs index |
| [Architecture](./docs/architecture/README.md) | System design and service boundaries |
| [Ownership Model](./docs/architecture/ownership.md) | End-to-end operating model for the platform |
| [ADRs](./docs/decisions/README.md) | Architecture decisions and tradeoffs |
| [RCAs](./docs/incidents/README.md) | Incidents, failures, and recovery notes |
| [Operations Notes](./docs/notes/README.md) | Runbooks and implementation notes |
| [Workflows](./docs/workflows.md) | CI/CD and GitOps workflow reality |
| [Visual Gallery](./docs/visual/README.md) | Dashboards and platform screenshots |

---

## 🛠️ Tech Stack

### Platform & Infrastructure

- Kubernetes (K3s), Helm, Docker
- Argo CD (GitOps)
- OpenTofu (Terraform alternative)

### Observability

- OpenTelemetry
- Prometheus, Grafana
- Loki (logs), Tempo (traces), Thanos (metrics scaling)

### Data & Storage

- PostgreSQL (CloudNativePG)
- MinIO (S3-compatible)
- Azure Blob Storage

### Networking & Security

- Cilium (eBPF networking)
- OpenBao (Secrets Management)
- Tailscale

### Languages

- Go (backend services)

---

## ⚠️ Challenges

One challenge was debugging service communication with Cilium networking.

- **Problem:** Services were unreachable even though pods were running  
- **Cause:** Incorrect network policies blocking traffic  
- **Fix:** Used logs and metrics to identify dropped packets and corrected policies  

---

## 🚀 Project Evolution

This platform evolved through multiple phases:

- **Foundations:** Docker, Go services, host-level visibility  
- **Kubernetes Migration:** Moved workloads to K3s + GitOps  
- **SRE Maturity:** Full observability (logs, metrics, traces)  
- **Infrastructure:** OpenTofu layered architecture  
- **Advanced Networking:** Cilium (eBPF)  
- **Operational Maturity:** Argo CD orchestration + HA systems  

👉 [View Full Evolution Log](https://victoriacheng15.github.io/observability-hub/evolution.html)

---

## 🚀 Getting Started

<details>
<summary><b>Local Setup</b></summary>

### Prerequisites

- Go
- K3s
- Helm
- Make
- Nix

### Setup

```bash
cp .env.example .env
```

### Deploy Infrastructure

```bash
cd tofu
tofu init
tofu apply
```

### Run Services

```bash
make proxy-build
make mcp-build
make install-services
```

### Verify

- Grafana: <http://localhost:30000>  
- Check logs via Loki  

</details>

---

## 📌 Summary

This project demonstrates how to build a production-like DevOps platform using:

- Kubernetes + GitOps  
- Full observability (logs, metrics, traces)  
- Infrastructure as Code  
- High availability systems  
- Real-world debugging and failure handling  

It reflects how modern platform teams operate in real environments.
