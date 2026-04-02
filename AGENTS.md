# Agent Guide for Observability Hub

This document is the **Primary Heuristic** for AI agents. It defines the systemic boundaries, operational standards, and institutional memory required to contribute effectively to the Observability Hub.

## 1. Systemic Architecture & Mental Model

Agents must distinguish between the two primary orchestration tiers to avoid "circular dependencies" and resource contention.

### 🌌 Hybrid Orchestration Layers

- **Host Tier (Systemd)**: Reserved for hardware-level telemetry, security gates, and GitOps reconciliation. Reliability here is critical for cluster recovery. Core logic is strictly encapsulated in `internal/` to ensure reusability and enforce project boundaries.
- **Cluster Tier (K3s)**: Handles scalable data services (Postgres, Loki, Prometheus, Grafana, Tempo, MinIO). Orchestrated via **OpenTofu (IaC)** in `tofu/`.

### 🏗️ Directory Map (Consolidated Monorepo)

- **`cmd/`**: Minimal entry points for services. Focuses on configuration and orchestration.
  - **`cmd/web/`**: Static site generator entry point.
  - **`cmd/proxy/`**: API Gateway and GitOps webhook listener entry point.
  - **`cmd/worker/`**: Unified one-shot execution engine for Analytics and Ingestion tasks.
- **`internal/`**: Private Implementation Layer. Enforces Go's internal package visibility rules.
- **`k3s/`**: Kubernetes manifests and Helm values for the data platform.
- **`makefiles/`**: Modular logic for the root automation layer.
- **`systemd/`**: Host-tier unit files for production service management.
- **`scripts/`**: Operational utilities (Traffic gen, ADR creation, Tailscale gate).
- **`docs/`**: Institutional memory. ADRs, Architecture, Incidents, and Notes.

## 2. Staff-Level Automation (`Makefile`)

The project uses a unified automation layer. **Always prefer `make` commands** as they handle environment wrapping (Nix) and consistency checks automatically.

### 🛠️ Core Commands

| Domain | Command | Description |
| :--- | :--- | :--- |
| **Governance** | `make adr` | Creates a new Architecture Decision Record. |
| **IaC** | `tofu plan` / `apply` | Manages K3s data services and infrastructure state. |
| **Quality** | `make lint` | Lints markdown and configuration files. |
| **Go Dev** | `make test` | Runs full test suite across the monorepo. |
| **Security** | `make vuln-scan` | Executes `govulncheck` for dependency auditing. |
| **K3s Ops** | `make build-postgres` | Builds and imports the custom PG image into K3s. |
| **Host Ops** | `make proxy-build` | Builds proxy server to `bin/` and restarts the service. |

## 3. Engineering Standards

### 🐹 Go (Backend)

- **Thin Main**: Entry points in `cmd/` must be minimal. Move all core domain logic to `internal/`.
- **Internal-First**: Shared libraries reside in `internal/` to prevent external logic leakage.
- **Environment Loading**: Always use `internal/env` for standardized `.env` discovery.
- **Observability**: Every service must emit structured JSON logs using `internal/telemetry`.
- **Telemetry**: All instrumentation must be handled through the centralized `internal/telemetry` library.
- **Testing**: Table-driven tests are the standard. Run `make go-cov` to verify coverage.

### 📝 Institutional Memory (Documentation)

- **ADRs (`docs/decisions/`)**: Mandatory for any architectural pivot.
- **RCA (`docs/incidents/`)**: Document every failure to prevent regression.
- **Golden Path**: Maintain `docs/workflows.md` to reflect the CI/CD reality.

## 4. Operational Excellence & Safety

- **Secrets**: NEVER commit secrets. Use `.env` for local dev and OpenBao for production.
- **GitOps**: Host-tier changes are applied via `gitops_sync.sh` (triggered by Proxy webhooks).
- **Security**: All Kubernetes manifests must pass `kube-lint`. All Go code must pass `go-vuln-scan`.
