# Agent Guide for Observability Hub

This document is the **Primary Heuristic** for AI agents. It defines the systemic boundaries, operational standards, and institutional memory required to contribute effectively to the Observability Hub.

## 1. Systemic Architecture & Mental Model

Agents must distinguish between the two primary orchestration tiers to avoid "circular dependencies" and resource contention.

### 🌌 Hybrid Orchestration Layers

- **Host Tier (Systemd)**: Reserved for hardware-level telemetry, security gates, and GitOps reconciliation. Reliability here is critical for cluster recovery. Core logic is extracted into `pkg/` libraries to ensure reusability and consistency across different execution triggers (CLI, API, and future AI tools).
- **Cluster Tier (K3s)**: Handles scalable data services (Postgres, Loki, Prometheus, Grafana, Tempo, MinIO). Orchestrated via IaC in `k3s/`.

### 📦 Distribution Pattern

To maintain a clean repository and ensure operational stability, all compiled binaries must be output to the root `dist/` directory. Systemd unit files and automated scripts should reference artifacts from this location.

### 🏗️ Directory Map

- **`pkg/`**: Shared Go modules (DB, brain, env, logger, metrics, secrets, telemetry). Maintain stable interfaces.
- **`services/`**: Standalone binaries and operational entry points.
  - **`services/proxy/`**: The \"Central Nervous System.\" API gateway and GitOps webhook listener.
  - **`services/system-metrics/`**: Host hardware telemetry collector.
  - **`services/second-brain/`**: Knowledge ingestion pipeline.
- **`dist/`**: Production artifacts. Centralized directory for all compiled binaries.
- **`k3s/`**: Kubernetes manifests and Helm values for the data platform.
- **`makefiles/`**: Modular logic for the root automation layer.
- **`systemd/`**: Host-tier unit files for production service management.
- **`page/`**: Static site generator for public-facing portfolio.
- **`scripts/`**: Operational utilities (Traffic gen, ADR creation, Tailscale gate).
- **`docs/`**: Institutional memory. ADRs, Architecture, Incidents, and Notes.

## 2. Staff-Level Automation (`Makefile`)

The project uses a unified automation layer. **Always prefer `make` commands** as they handle environment wrapping (Nix) and consistency checks automatically.

### 🛠️ Core Commands

| Domain | Command | Description |
| :--- | :--- | :--- |
| **Governance** | `make adr` | Creates a new Architecture Decision Record. |
| **Quality** | `make lint` | Lints markdown and configuration files (`lint-configs`). |
| **Go Dev** | `make go-test` | Runs full test suite across the monorepo. |
| **Security** | `make go-vuln-scan` | Executes `govulncheck` for dependency auditing. |
| **K3s Ops** | `make kube-lint` | Validates K8s manifests for security violations. |
| **Host Ops** | `make reload-services` | Safely reloads host-tier systemd units. |

## 3. Engineering Standards

### 🐹 Go (Backend)

- **Library-First**: Move core domain logic to `pkg/` before implementing the service entry point. Services should be thin wrappers around library capabilities.
- **Environment Loading**: Always use `pkg/env` for standardized `.env` discovery. Do not use `godotenv` directly in services.
- **Dependency Management**: Delegate driver registration (e.g., `lib/pq`) to `pkg/db` to avoid redundant blank imports in services.
- **Failure Modes**: Never swallow errors. Use explicit wrapping: `fmt.Errorf(\"context: %w\", err)`.
- **Observability**: Every service must emit JSON-formatted logs to `stdout` using `pkg/logger`.
- **Telemetry**: All instrumentation must be handled through the centralized `pkg/telemetry` library.
- **Testing**: Table-driven tests are the standard. Run `make go-cov` to verify coverage. Maintain a minimum of 80% coverage for `pkg/` libraries.

### 🎨 HTML/CSS (Frontend)

- **Zero Frameworks**: Use native HTML5 and CSS3 only.
- **Styling**: Leverage CSS variables in `:root` for dark-theme consistency.

### 📝 Institutional Memory (Documentation)

- **ADRs (`docs/decisions/`)**: Mandatory for any architectural pivot.
- **RCA (`docs/incidents/`)**: Document every failure to prevent regression.
- **Golden Path**: Maintain `docs/workflows.md` to reflect the CI/CD reality.

## 4. Operational Excellence & Safety

- **Secrets**: NEVER commit secrets. Use `.env` for local dev and OpenBao for production secrets.
- **GitOps**: Host-tier changes are applied via `gitops_sync.sh` (triggered by Proxy webhooks).
- **Observability**: Any new service must be integrated into the telemetry pipeline (Logs to Loki, Metrics to Postgres/Prometheus).
- **Security**: All Kubernetes manifests must pass `kube-lint`. All Go code must pass `go-vuln-scan`.

## 5. Failure Mode Analysis (FMA)

Before proposing a change, agents should ask:
1. "Does this create a circular dependency between the host and the cluster?"
2. "How will this be debugged in production if the network is down?"
3. "Is this change recorded in an ADR to preserve the 'Why'?"
