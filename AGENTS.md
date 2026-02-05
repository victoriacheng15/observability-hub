# Agent Guide for Observability Hub

This document provides context and instructions for AI agents working on the **Observability Hub** project, a Go-based observability platform and mentorship sandbox.

## 1. Project Overview

**Observability Hub** is a modular observability platform designed for hybrid and edge environments. It bridges local `systemd` services with cloud-native patterns (Loki, Alloy, Postgres) orchestrated via Kubernetes.

- **Core Tech**: Go (Golang), PostgreSQL, Loki, Grafana Alloy, Systemd, Nix, Kubernetes (k3s).
- **Styling**: Native HTML/CSS (Dark Theme via CSS Variables).
- **Architecture**: Modular Monorepo (`pkg/`, `proxy/`, `system-metrics/`, `page/`).

## 2. Build and Test Commands

The project relies on **Nix** for a reproducible environment. **Always use the `make <target>` (which auto-wraps in Nix) or `make nix-<target>` variants** for Go-related tasks.

### Project & Documentation

| Command | Description |
| :--- | :--- |
| `make adr` | **Primary ADR Command**. Creates a new Architecture Decision Record (ADR). |
| `make lint` | Lints and fixes styling in markdown files. |

### Go Development (Nix-wrapped)

| Command | Description |
| :--- | :--- |
| `make go-format` | Automatically formats and simplifies Go code inside `nix-shell`. |
| `make go-test` | Runs all Go unit tests across the monorepo inside `nix-shell`. |
| `make go-update` | Updates all Go dependencies and runs `go mod tidy` inside `nix-shell`. |
| `make go-lint` | Runs static analysis (go vet) across all modules inside `nix-shell`. |
| `make go-cov` | Generates and displays test coverage reports inside `nix-shell`. |
| `make page-build` | Builds the GitHub Page generator and refreshes static assets. |
| `make metrics-build` | Builds and restarts the host-level `system-metrics` collector. |
| `make proxy-build` | Rebuilds and restarts the `proxy` API gateway. |

### Host Tier (Systemd & Secrets)

| Command | Description |
| :--- | :--- |
| `make install-services` | Symlinks and enables all `systemd` units for the host tier. |
| `make reload-services` | Reloads `systemd` configurations from the repository. |
| `make uninstall-services` | Completely stops and removes all project-related systemd units. |
| `make bao-status` | Checks the health and seal status of the OpenBao secret store. |

### Kubernetes Tier (k3s)

| Command | Description |
| :--- | :--- |
| `make k3s-status` | Overview of all resources in the cluster `observability` namespace. |
| `make k3s-alloy-up` | Deploy or rollout restart Grafana Alloy in the cluster. |
| `make k3s-loki-up` | Deploy or rollout restart Loki in the cluster. |
| `make k3s-grafana-up` | Deploy or rollout restart Grafana in the cluster. |
| `make k3s-postgres-up` | Deploy or rollout restart PostgreSQL in the cluster. |
| `make k3s-backup-<name>` | Safely backup a cluster resource (e.g., `make k3s-backup-postgres`). |

## 3. Code Style Guidelines

### Go (Backend)

- **Strict Adherence**: Code must pass `gofmt` and strict linting.
- **Testing**: **Table-Driven Tests** are preferred. Use the standard library `testing` package.
- **Error Handling**: **Explicit, wrapped errors** (e.g., `fmt.Errorf("failed to connect: %w", err)`). Do not swallow errors.

### HTML/CSS (Frontend)

- **Frameworks**: None. Native HTML/CSS only.
- **Styling**: Use CSS Variables defined in `:root` (Dark Theme). Layouts via CSS Grid and Flexbox.
- **Structure**: Semantic HTML5 (header, nav, main, footer).

### Documentation

- **ADRs**: Architectural decisions (`docs/decisions`) must strictly follow the ADR format.
- **Freshness**: Maintain `README.md` clarity and ensure documentation matches implementation.

## 4. Testing Instructions

- **Unit Tests**: Run `make nix-go-test` to execute the standard Go test suite.
- **Coverage**: Run `make nix-go-cov` to generate coverage reports in the terminal.
- **New Features**: Any new logic must include accompanying table-driven unit tests.

## 5. Security & Automation

- **Infrastructure**:
  - **`k3s/`**: IaC for Kubernetes-native services (Loki, Postgres, Alloy, Grafana).
  - **`systemd/`**: Production service definitions for host-level management.
- **Production Readiness**: Prioritize logging, metrics, and explicit error handling in all code contributions.
- **Secrets**: Never commit secrets. Ensure `.env` is used for sensitive configuration.
