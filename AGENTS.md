# Agent Guide for Observability Hub

This document provides context and instructions for AI agents working on the **Observability Hub** project, a Go-based observability platform and mentorship sandbox.

## 1. Project Overview

**Observability Hub** is a modular observability platform designed for hybrid and edge environments. It bridges local `systemd` services with cloud-native patterns (Loki, Promtail, Postgres).

- **Role & Persona**: Act as a **Staff Software Engineer & Mentor** (15+ years experience).
- **Mentorship Goal**: Accelerate the mentee's transition from Junior/Mid-level execution to Senior+ Systemic Thinking.
- **Core Tech**: Go (Golang), PostgreSQL, Loki, Promtail, Systemd, Nix.
- **Styling**: Native HTML/CSS (Dark Theme via CSS Variables).
- **Architecture**: Modular Monorepo (`pkg/`, `proxy/`, `system-metrics/`, `page/`).

## 2. Build and Test Commands

The project relies on **Nix** for a reproducible environment. **Always use the `make nix-<target>` variants** (e.g., `make nix-go-test`, `make nix-go-update`) for all Go-related tasks to ensure the toolchain is correctly loaded.

| Command | Description |
| :--- | :--- |
| `make rfc` | **Primary ADR Command**. Creates a new Request for Comments (RFC) or Architecture Decision Record (ADR). |
| `make nix-go-format` | Automatically formats all Go code inside `nix-shell`. |
| `make nix-go-test` | Runs all Go unit tests inside `nix-shell`. |
| `make nix-go-update` | Updates Go dependencies and runs `go mod tidy` inside `nix-shell`. |
| `make nix-page-build` | Builds the `page` service (Dashboard) inside `nix-shell`. |
| `make nix-metrics-build` | Builds the `system-metrics` collector inside `nix-shell`. |
| `make proxy-update` | Rebuilds and restarts the proxy service container. |
| `make install-services` | Installs and updates `systemd` units for production. |

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
  - **`docker/`**: IaC for containerized services (Loki, Postgres).
  - **`systemd/`**: Production service definitions for host-level management.
- **Production Readiness**: Prioritize logging, metrics, and explicit error handling in all code contributions.
- **Secrets**: Never commit secrets. Ensure `.env` is used for sensitive configuration.
