# Self-Hosted Observability Hub

A resilient and reliability-focused unified telemetry platform architected to demonstrate SRE & Platform Engineering principles: full-stack observability, GitOps-driven infrastructure, and standardized data ingestion. It unifies system metrics, application events, and logs into a single queryable layer using PostgreSQL (TimescaleDB) and Loki, visualized via Grafana, all orchestrated within a **Kubernetes (k3s)** environment.

---

## 🌐 Live Site

[Explore Live Telemetry & System Evolution](https://victoriacheng15.github.io/observability-hub/)

---

## 🏗️ Engineering Principles

- **Unified Observability:** Correlation of infrastructure telemetry and application business events into a single, queryable plane. Full-stack visibility is the default state, ensuring all services are observed via a consistent, unified standard.
- **Platform Abstraction:** Decoupling of data ingestion from storage engines. Standardized APIs provide stable interfaces for clients, allowing the underlying pipeline logic and database schemas to evolve without disrupting upstream producers.
- **GitOps & State Convergence:** Enforcement of configuration consistency between version control and the running environment. Automated reconciliation engines detect and correct drift, ensuring the "Source of Truth" is always the reality.
- **Hybrid Orchestration:** Strategic deployment utilizing the most effective primitives for the task. It combines **Kubernetes (k3s)** isolation for core data services with native host performance (Systemd) for critical automation and hardware-level telemetry.

---

## 🛠️ Tech Stack

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![MongoDB](https://img.shields.io/badge/MongoDB-%234ea94b.svg?style=for-the-badge&logo=mongodb&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Kubernetes (k3s)](https://img.shields.io/badge/Kubernetes-326CE5.svg?style=for-the-badge&logo=Kubernetes&logoColor=white)
![Helm](https://img.shields.io/badge/Helm-0F1689.svg?style=for-the-badge&logo=Helm&logoColor=white)
![Grafana](https://img.shields.io/badge/grafana-%23F46800.svg?style=for-the-badge&logo=grafana&logoColor=white)

---

## 📚 Architectural Approach & Documentation

This section provides a deeper look into the system's structure, components, and data flow.

### System Architecture Diagram

This diagram shows the high-level flow of data from collection to visualization.

```mermaid
flowchart LR
    subgraph External ["External"]
        GH(GitHub Webhooks)
        Apps
        Mongo(MongoDB Atlas)
    end

    subgraph HostServices ["Native Host Services (Systemd)"]
        Proxy[Proxy API & GitOps Trigger]
        Gate[Tailscale Gate]
        Metrics[Metrics Collector]
        Bao[OpenBao Secret Store]
    end

    subgraph DataPlatform ["Data Infrastructure (Kubernetes)"]
        PG(PostgreSQL)
        A(Grafana Alloy)
        L(Loki)
        G(Grafana)
    end

    %% Data Pipeline
    GH -- Webhook --> Proxy
    Proxy -- Sync --> GH
    Apps -- Events --> Mongo
    Mongo -- Pull --> Proxy
    Proxy -- Write --> PG
    Metrics -- Telemetry --> PG
    Bao -.->|Secrets| Proxy
    Bao -.->|Secrets| Metrics
    PG --> G

    %% Logging Pipeline
    Proxy -- Logs --> A
    Gate -- Logs --> A
    Metrics -- Logs --> A
    A -- Scrape Journal --> L
    L --> G
```

### Component Breakdown

This table lists the main services and components within the observability hub, along with their responsibilities and location within the repository.

| Service / Component | Responsibility | Location |
| :------------------ | :------------- | :------- |
| **proxy** | Native Go service acting as an API gateway, Data Pipeline engine, and **GitOps Webhook listener**. | `proxy/` |
| **tailscale-gate** | Security agent managing public access (Tailscale Funnel) based on Proxy health. | `scripts/` |
| **system-metrics** | Lightweight Go collector for host CPU, memory, disk, and network stats. | `system-metrics/` |
| **openbao** | Centralized, encrypted secret storage and management. | `systemd/` |
| **page** | Go static-site generator for the public-facing portfolio page. | `page/` |
| **PostgreSQL** | Primary relational storage (TimescaleDB + PostGIS) managed as a **Kubernetes StatefulSet**. | `k3s/postgres/` |
| **Grafana** | Primary visualization and dashboarding tool deployed in **Kubernetes**. | `k3s/grafana/` |
| **Loki** | Log aggregation system for all services running in **Kubernetes**. | `k3s/loki/` |
| **Grafana Alloy** | Unified telemetry agent (Kubernetes DaemonSet) for host journal collection. | `k3s/alloy/` |
| **gitops-sync** | Reconciliation script triggered by the Proxy to enforce repository state. | `scripts/` |
| **reading-sync** | Systemd service that periodically triggers the `proxy` Data Pipeline. | `systemd/` |

### External Dependencies

These components exist outside this repository but are integral to the data pipeline:

| Dependency | Role |
| :--- | :--- |
| **Client Applications** | Sources of event data (e.g., Cover Craft, Personal Reading Analytics). |
| **MongoDB Atlas** | Interim cloud storage used as a buffer/queue for external event logs. |

### Data Flow

The system categorizes data flow into three main streams, correlating events, cluster health, and host stability.

1. **Application Events:**
    - **Source:** Client Applications (e.g., Cover Craft, Personal Reading Analytics Dashboard) write events to MongoDB Atlas.
    - **Process:** The reading-sync service (Systemd) triggers the proxy to fetch, transform, and persist records into PostgreSQL.
    - **Dashboard:** Reading Analytics.
2. **Kubernetes Monitoring:**
    - **Source:** Orchestrated services (PostgreSQL, Loki, Grafana, Alloy).
    - **Collection:** Grafana Alloy scrapes cluster metadata and internal service metrics.
    - **Dashboard:** Cluster Health.
3. **Systemd Monitoring:**
    - **Source:** Host services and hardware telemetry.
    - **Collection:** The system-metrics collector (automated via Systemd timer) flushes hardware stats to PostgreSQL, while **Grafana Alloy** scrapes journald for service logs.
    - **Dashboards:** Systemd Monitoring and Homelab (hardware metrics).

For deep dives into the system's inner workings, operational guides, and decision logs:

- **[Documentation Hub](./docs/README.md)**: Central entry point for Architecture, Decisions (ADRs), and Operational Notes.

---

## 🚢 Deployment Strategy

The platform employs a Hybrid Deployment Model to balance security, reliability, and performance:

### 1. Event-Driven GitOps (Webhook-based)

Critical observability infrastructure and host configurations are managed by an automated reconciliation workflow.

- **Mechanism:** GitHub sends a **Webhook** event (Push or PR Merge) to the Proxy service.
- **Action:** The Proxy validates the request signature and executes the local `gitops_sync.sh` script to update the repository and reload services.
- **Benefit:** Real-time updates and improved security by ensuring the public entry point (via Tailscale Funnel) is managed dynamically based on system state.

### 2. Public Portfolio (Push-based CI/CD)

The static status page is built and deployed via GitHub Actions.

- **Mechanism:** Standard CI pipeline defined in `.github/workflows/deploy.yml`.
- **Action:** Builds the Go `page` generator and deploys the output to GitHub Pages.
- **Benefit:** Fast feedback loops and high availability for the public-facing component.

---

## 🚀 Getting Started (Local Development)

This guide will help you set up and run the `observability-hub` locally using **Kubernetes (k3s)**.

### Prerequisites

Ensure you have the following installed on your system:

- [Go](https://go.dev/doc/install) (version 1.21 or newer)
- [k3s](https://k3s.io/) (Lightweight Kubernetes)
- [Helm](https://helm.sh/)
- `make` (GNU Make)
- [Nix](https://nixos.org/download.html) (for reproducible toolchains)

### 1. Configuration

The project uses a `.env` file to manage environment variables, especially for database connections and API keys.

```bash
# Start by copying the example file
cp .env.example .env
```

You will need to edit the newly created `.env` file to configure connections for MongoDB Atlas, PostgreSQL (k3s NodePort), and other services.

### 2. Build and Run the Stack

The cluster resources are managed via the root **Makefile**. Deploy the entire stack into the `observability` namespace:

```bash
# Deploy all core components to k3s
make k3s-alloy-up
make k3s-loki-up
make k3s-grafana-up
make k3s-postgres-up
```

These commands will:

- Template manifests from Helm charts and local values.
- Apply configurations to the cluster.
- Perform a rollout restart to ensure the latest state is active.

To view the cluster status:

```bash
make k3s-status
```

### 3. Verification

Once the pods are in a `Running` state, you can verify their functionality:

- **Grafana Dashboards:** Access Grafana at `http://localhost:30000`.
  - Default login: `admin` / (Retrieved via `kubectl get secret`)
  - You should see your provisioned data sources and dashboards.
- **Static Portfolio Site:** The `page` service builds your public portfolio site into the `page/dist` directory. You can inspect the generated static HTML files there.

### 4. Managing the Cluster

To stop or remove resources, use the standard `kubectl delete` commands targeting the `observability` namespace.
