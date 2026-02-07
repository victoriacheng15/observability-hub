# Architectural Decision Records (ADR)

This directory serves as the **Institutional Memory** for the Observability Hub. It documents the "Why" behind major technical choices, ensuring the project remains maintainable and its evolution is transparent.

---

## 📂 Decision Log

| ADR | Title | Status |
| :--- | :--- | :--- |
| **013** | [Standardize on OpenTelemetry](./013-standardize-on-opentelemetry.md) | 🟢 Proposed |
| **012** | [Migrate Promtail to Alloy](./012-migrate-promtail-to-alloy.md) | 🔵 Accepted |
| **011** | [Phased Migration Strategy to K3s](./011-phased-k3s-migration-strategy.md) | 🔵 Accepted |
| **010** | [Integrate OpenBao](./010-integrate-openbao.md) | 🔵 Accepted |
| **009** | [Standardized Database Connection Methods](./009-standardized-db-connection-methods.md) | 🔵 Accepted |
| **008** | [GitOps via Proxy Webhook](./008-gitops-via-proxy-webhook.md) | 🔵 Accepted |
| **007** | [k3s Shadow Deployment & Orchestration](./007-k3s-shadow-deployment-orchestration.md) | 🔵 Accepted |
| **006** | [Shared Database Configuration Module](./006-shared-database-module.md) | 🔵 Accepted |
| **005** | [Centralized GitOps Reconciliation Engine](./005-gitops-reconciliation-engine.md) | 🟡 Superseded |
| **004** | [Spatial Keyboard Telemetry Pipeline](./004-spatial-telemetry-keyboard.md) | 🟡 Superseded |
| **003** | [Shared Structured Logging Library](./003-shared-logging-library.md) | 🔵 Accepted |
| **002** | [Cloud-to-Homelab Telemetry Bridge](./002-cloud-to-local-bridge.md) | 🔵 Accepted |
| **001** | [PostgreSQL vs. InfluxDB for Metrics Storage](./001-postgres-vs-influxdb.md) | 🔵 Accepted |

---

## 🛠️ Process & Standards

This section defines how we propose, evaluate, and document architectural changes.

### Decision Lifecycle

| Status | Meaning |
| :--- | :--- |
| **🟢 Proposed** | Planning phase. The design is being discussed or researched. |
| **🔵 Accepted** | Implementation phase or completed. This is the current project standard. |
| **🟡 Superseded** | Historical record. This decision has been replaced by a newer ADR. |

### Conventions

- **File Naming:** `00X-descriptive-title.md`
- **Dates:** Use ISO 8601 format (`YYYY-MM-DD`).
- **Formatting:** Use hyphens (`-`) for all lists; no numbered lists.
- **Automation:** Run `make rfc` to interactively generate a new file that follows these standards.

### 📝 ADR Template

To create a new proposal, copy the block below into a new `.md` file.

```markdown
# ADR [00X]: [Descriptive Title]

- **Status:** Proposed | Accepted | Superseded
- **Date:** YYYY-MM-DD
- **Author:** Victoria Cheng

## Context and Problem Statement

What specific issue triggered this change?

## Decision Outcome

What was the chosen architectural path?

## Consequences

- **Positive:** (e.g., Faster development, resolved dependency drift).
- **Negative/Trade-offs:** (e.g., Added complexity to the CI/CD pipeline).

## Verification

- [ ] **Manual Check:** (e.g., Verified logs/UI locally).
- [ ] **Automated Tests:** (e.g., `make nix-go-test` passed).
```
