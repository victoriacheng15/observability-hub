# Observability Hub Architecture

This directory contains the architectural blueprints for the Observability Hub. The system follows a hybrid model: **Kubernetes (k3s)** runs the observability data platform, while native host services handle hardware-aware automation, ingress, and local control.

## Architecture At A Glance

The Observability Hub is a self-hosted platform that collects telemetry from local services, routes it through a unified observability pipeline, and exposes the results through dashboards, traces, logs, and operator tooling.

For a fast mental model, the main system story is:

`Services and workloads -> OpenTelemetry -> Loki / Tempo / Prometheus -> Grafana`

The surrounding platform supports that flow:

- **Proxy**: Host-level API gateway and GitOps webhook listener
- **Worker**: Cluster-native batch engine for analytics and ingestion jobs
- **OpenTelemetry Collector**: Central intake and routing point for logs, metrics, and traces
- **LGTM Stack**: Loki, Tempo, and Prometheus as the main observability backends
- **Grafana**: Unified visualization layer for operators
- **ArgoCD + OpenTofu**: Deployment and reconciliation control plane

If you are reading this as a recruiter or hiring manager, start with the pages that explain the system story first, then drill into implementation details as needed.

## Suggested Reading Path

- **Start here:** [Observability](./core-concepts/observability.md) for the end-to-end telemetry flow
- **Then read:** [Deployment Model](./infrastructure/deployment.md) for how the platform runs locally
- **Then read:** [Proxy Service](./services/proxy.md) and [Unified Worker](./services/worker.md) for the main custom services
- **Optional deep dives:** Automation, security, MCP, and simulation components

---

## 📂 Documentation Domains

### 🧠 [Core Concepts](./core-concepts/)

Fundamental patterns and cross-cutting concerns that define how the system operates.

- **[Automation & GitOps](./core-concepts/automation.md)**: Declarative reconciliation via ArgoCD and event-driven self-healing patterns.
- **[Observability](./core-concepts/observability.md)**: Standards for JSON logging and OpenTelemetry pipelines.

### 🏗️ [Infrastructure](./infrastructure/)

The runtime environment and foundational deployment strategies.

- **[Deployment Model](./infrastructure/deployment.md)**: Details on the hybrid Kubernetes/Systemd orchestration.
- **[Security](./infrastructure/security.md)**: Tailscale Funnel gating, HMAC authentication, and isolation boundaries.

### ⚙️ [Services](./services/)

Deep dives into the logic and implementation of specific system components.

- **[Hardware Simulation](./services/hardware-sim.md)**: Fleet of synthetic sensors and chaos injection engine.
- **[MCP Servers](./services/mcp-servers.md)**: The "Agentic Interface" suite for autonomous operations.
- **[Proxy Service](./services/proxy.md)**: The API Gateway and GitOps listener.
- **[Tailscale Gate](./services/tailscale-gate.md)**: Logic for the automated funnel gatekeeper.
- **[Unified Worker](./services/worker.md)**: The one-shot execution engine for Analytics and Ingestion tasks.
