# Observability Hub Architecture

This directory contains the detailed architectural blueprints for the Observability Hub. The system follows a hybrid model, utilizing **OpenTofu** to manage **Kubernetes (k3s)** core data services and native Systemd units for host-level automation and data pipelines.

---

## 📂 Documentation Domains

### 🧠 [Core Concepts](./core-concepts/)

Fundamental patterns and cross-cutting concerns that define how the system operates.

- **[Automation & GitOps](./core-concepts/automation.md)**: Webhook-driven reconciliation and self-healing patterns.
- **[Observability](./core-concepts/observability.md)**: Standards for JSON logging and OpenTelemetry pipelines.

### 🏗️ [Infrastructure](./infrastructure/)

The runtime environment and foundational deployment strategies.

- **[Deployment Model](./infrastructure/deployment.md)**: Details on the hybrid Kubernetes/Systemd orchestration.
- **[Security](./infrastructure/security.md)**: Tailscale Funnel gating, HMAC authentication, and isolation boundaries.

### ⚙️ [Services](./services/)

Deep dives into the logic and implementation of specific system components.

- **[Collectors Service](./services/collectors.md)**: Host-level telemetry collector for metrics and Tailscale status.
- **[Proxy Service](./services/proxy.md)**: The API Gateway and GitOps listener.
- **[Ingestion Service](./services/ingestion.md)**: Unified data orchestration engine (Reading Analytics + Second Brain).
- **[MCP Servers](./services/mcp-servers.md)**: The "Agentic Interface" suite for autonomous operations.
- **[Tailscale Gate](./services/tailscale-gate.md)**: Logic for the automated funnel gatekeeper.
