# Self-Hosted Observability Hub

A resilient and reliability-focused unified telemetry platform architected to demonstrate SRE & Platform Engineering principles: full-stack observability, GitOps-driven infrastructure, and standardized data ingestion. It unifies system metrics, application events, and logs into a single queryable layer using **PostgreSQL (TimescaleDB)** and **Grafana Loki**, visualized via **Grafana**, all orchestrated within a **Kubernetes (K3s)** environment.

---

## 🌐 Live Site

[Explore Live Telemetry & System Evolution](https://victoriacheng15.github.io/observability-hub/)

---

## 🏗️ Engineering Principles

- **Observability-First:** Full-stack visibility is foundational. Every service implements advanced signals (lag, saturation, pool health) as a project standard.
- **Infrastructure Abstraction:** Decoupling plumbing from logic. Shared "Pure Wrappers" handle connection and OTel complexity, allowing services to focus strictly on domain value.
- **GitOps & State Convergence:** Configuration as code with automated reconciliation. Version control is the ultimate source of truth for the environment state.
- **Hybrid Orchestration:** Utilizing Kubernetes for data persistence and native Systemd for host-level automation.

---

## 🛠️ Tech Stack

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Nix](https://img.shields.io/badge/Nix-5277C3?style=for-the-badge&logo=NixOS&logoColor=white)

![Kubernetes (K3s)](https://img.shields.io/badge/Kubernetes-326CE5.svg?style=for-the-badge&logo=Kubernetes&logoColor=white)
![Helm](https://img.shields.io/badge/Helm-0F1689.svg?style=for-the-badge&logo=Helm&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![OpenBao](https://img.shields.io/badge/OpenBao-6d7174?style=for-the-badge&logo=openbao&logoColor=white)
![Tailscale](https://img.shields.io/badge/Tailscale-%235d21d0.svg?style=for-the-badge&logo=tailscale&logoColor=white)

![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-%23000000.svg?style=for-the-badge&logo=opentelemetry&logoColor=white)
![Grafana Loki](https://img.shields.io/badge/Loki-%23F46800.svg?style=for-the-badge&logo=grafana&logoColor=white)
![Grafana](https://img.shields.io/badge/grafana-%23F46800.svg?style=for-the-badge&logo=grafana&logoColor=white)
![Grafana Tempo](https://img.shields.io/badge/Tempo-%23F46800.svg?style=for-the-badge&logo=grafana&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)

![PostgreSQL](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![MinIO (S3)](https://img.shields.io/badge/MinIO-be172d?style=for-the-badge&logo=minio&logoColor=white)
![MongoDB](https://img.shields.io/badge/MongoDB-%234ea94b.svg?style=for-the-badge&logo=mongodb&logoColor=white)

---

## 📚 Architectural Approach & Documentation

This section provides a deeper look into the system's structure, components, and data flow.

### System Architecture Diagram

This diagram shows the high-level flow of data from collection to visualization, highlighting the hybrid orchestration between host services and the Kubernetes data platform.

```mermaid
flowchart TB
    subgraph ObservabilityHub ["Observability Hub"]
        direction TB
        subgraph Logic ["Data Ingestion"]
            subgraph External ["External Sources"]
                GH(GitHub Webhooks/Journals)
                Mongo(MongoDB Atlas)
            end

            subgraph Security [Security]
                Bao[OpenBao]
                Tailscale[Tailscale]
            end

            GoApps["Go Services (Proxy, Reading Sync, Second Brain)"]
            Collectors["Collectors (Host Metrics & Tailscale)"]
        end

        OTEL[OpenTelemetry Collector]

        Observability["Loki, Tempo, and Prometheus (Thanos)"]
        subgraph Storage ["Data Engines"]
            PG[(PostgreSQL)]
            S3[(MinIO - S3)]
        end
        

        subgraph Visualization ["Visualization"]
            Grafana[Grafana Dashboards]
        end
    end

    %% Data Pipeline Connections
    GH --> GoApps
    Mongo --> GoApps
    Observability -- "Host Metrics" --> Collectors
    Tailscale -- "Status" --> Collectors
    Collectors -- "Host Metrics Data" --> PG
    GoApps -- Data --> PG

    %% Telemetry Pipeline (OTLP)
    GoApps & Collectors -- "Logs, Metrics, Traces" --> OTEL
    OTEL --> Observability
    Observability -- "Offload" --> S3

    %% Visualization Connections
    Observability & PG --> Grafana
```

### Component Breakdown

The platform is split into two logical layers: **Native Host Services** for automation and hardware-level telemetry, and **Data Infrastructure** for scalable storage and visualization.

#### Native Host Services

| Service / Component | Responsibility | Location |
| :------------------ | :------------- | :------- |
| **gitops-sync** | Reconciliation script for automated state enforcement. | `scripts/` |
| **openbao** | Centralized, encrypted secret storage and management. | `systemd/` |
| **page** | Go static-site generator for the public-facing portfolio page. | `page/` |
| **proxy** | API gateway and **GitOps Webhook listener**. | `services/proxy/` |
| **reading-sync** | Automated data pipeline syncing MongoDB data to local PostgreSQL. | `services/reading-sync/` |
| **second-brain** | Ingests atomic thoughts from GitHub journals into PostgreSQL. | `services/second-brain/` |
| **tailscale-gate** | Security agent managing public access (Tailscale Funnel) based on Proxy health. | `scripts/` |

#### Data Infrastructure (Kubernetes)

| Service / Component | Responsibility | Location |
| :------------------ | :------------- | :------- |
| **Collectors** | Collects Tailscale data from hosts. | `k3s/collectors/` |
| **Grafana** | Centralized visualization and dashboarding platform. | `k3s/grafana/` |
| **Grafana Loki** | Log aggregation and query system for the entire stack. | `k3s/loki/` |
| **MinIO** | S3-compatible object storage for long-term trace and log persistence. | `k3s/minio/` |
| **OpenTelemetry Collector** | Standalone collector for multi-signal telemetry routing. | `k3s/opentelemetry/` |
| **PostgreSQL** | Primary relational storage (TimescaleDB + PostGIS) for metrics and events. | `k3s/postgres/` |
| **Prometheus** | Metrics storage, service discovery, and alerting engine. | `k3s/prometheus/` |
| **Grafana Tempo** | Distributed tracing backend for request correlation. | `k3s/tempo/` |
| **Thanos Store** | Query gateway for historical metrics stored in MinIO. | `k3s/thanos/` |

### External Dependencies

These components exist outside this repository but are integral to the data pipeline:

| Dependency | Role |
| :--- | :--- |
| **Client Applications** | Sources of event data (e.g., Cover Craft, Personal Reading Analytics). |
| **GitHub** | Source of webhooks for GitOps and issues for knowledge ingestion. |
| **MongoDB Atlas** | Interim cloud storage used as a buffer/queue for external event logs. |

For deep dives into the system's inner workings, operational guides, and decision logs:

- **[Documentation Hub](./docs/README.md)**: Central entry point for Architecture, Decisions (ADRs), and Operational Notes.

---

## 🚀 Getting Started (Local Development)

This guide will help you set up and run the `observability-hub` locally using **Kubernetes (K3s)**.

### Prerequisites

Ensure you have the following installed on your system:

- [Go](https://go.dev/doc/install) (version 1.25 or newer)
- [K3s](https://k3s.io/) (Lightweight Kubernetes)
- [Helm](https://helm.sh/)
- `make` (GNU Make)
- [Nix](https://nixos.org/download.html) (for reproducible toolchains)

### 1. Configuration

The project uses a `.env` file to manage environment variables, especially for database connections and API keys.

```bash
# Start by copying the example file
cp .env.example .env
```

You will need to edit the newly created `.env` file to configure connections for MongoDB Atlas, PostgreSQL (K3s NodePort), and other services.

### 2. Build and Run the Stack

The platform utilizes a hybrid orchestration model. You must deploy both the Kubernetes data tier and the native host services.

#### A. Data Infrastructure (K3s)

Deploy the observability backend into the `observability` namespace:

```bash
# Deploy core data and telemetry services
make k3s-postgres-up
make k3s-loki-up
make k3s-tempo-up
make k3s-prometheus-up
make k3s-grafana-up

# Deploy telemetry collectors
make k3s-otel-up
```

#### B. Native Host Services

Build and initialize the automation and telemetry collectors on the host:

```bash
# Build Go binaries
make proxy-build
make reading-build

# Install and start Systemd services (requires sudo)
make install-services

# Run Second Brain sync manually
make brain-sync
```

### 3. Verification

Once the stack is running, you can verify the end-to-end telemetry flow:

- **Cluster Health:** Access Grafana at `http://localhost:30000` (NodePort).
- **Service Logs:** Check logs for host components via Grafana Loki.

- **Knowledge Sync:** Manually trigger a Second Brain ingestion with `make brain-sync`.

### 4. Managing the Cluster

To stop or remove resources, use the standard `kubectl delete` commands targeting the `observability` namespace.
