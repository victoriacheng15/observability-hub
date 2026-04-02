# Observability Hub Architecture

This directory contains the detailed architectural blueprints for the Observability Hub. The system follows a hybrid model, utilizing **OpenTofu** to manage **Kubernetes (k3s)** core data services and native Systemd units for host-level automation and data pipelines.

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
