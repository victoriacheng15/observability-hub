# ADR 011: Phased Migration Strategy to K3s

- **Status:** Accepted
- **Date:** 2026-02-01
- **Author:** Victoria Cheng

## Context and Problem Statement

Building on the shadow deployment strategy established in ADR 007, the objective is to fully migrate the observability stack from Docker Compose to Kubernetes (k3s) to improve resilience and scalability. However, the stack contains critical stateful components (PostgreSQL, Loki) and active telemetry agents. A "Big Bang" migration (moving everything at once) carries significant risk of data loss, service interruption, and complex debugging if multiple components fail simultaneously.

A structured approach to migrate services is required to prioritize system stability and data integrity.

## Decision Outcome

Adopt a **Risk-Based Phased Migration Strategy**, moving components from "Lowest Risk" to "Highest Risk".

### The Migration Sequence

- **Phase 1: The Agent (Alloy)**
  - **Risk:** Low. If a failure occurs, only live logs are lost; historical data remains safe.
  - **Goal:** Establish the telemetry pipeline in k3s by deploying Grafana Alloy to take over systemd journal log collection from Promtail.
  - **Status:** Completed on 2026-02-02. Alloy is successfully collecting systemd journal logs and forwarding them to Docker-Loki.
- **Phase 2: The Log Store (Loki)**
  - **Risk:** Medium. Requires moving log data.
  - **Goal:** Establish persistent storage (PVCs) in k3s. Verify K3s networking.
  - **Status:** Completed on 2026-02-03. Loki is running in k3s with 10Gi persistence. Historical data migrated from Docker. Alloy updated to use internal K3s-Loki.
- **Phase 3: The UI (Grafana)**
  - **Risk:** Low/Medium. No critical state (dashboards can be re-imported).
  - **Goal:** Switch the "Pane of Glass" to run natively in the cluster.
  - **Status:** Completed on 2026-02-04. Grafana is running in k3s with 10Gi persistence. Dashboards, users, and plugins migrated from Docker.
- **Phase 4: The Core Data (PostgreSQL)**
  - **Risk:** Critical. The "Heart" of the system.
  - **Goal:** Migrate the relational database using "Dump & Restore" only after Phases 1-3 are stable.

## Consequences

### Positive

- **Risk Mitigation:** Isolates failures. If Phase 1 fails, rollbacks occur without touching the database.
- **Learning Curve:** Provides an opportunity to master k3s networking and storage on "safe" components (Alloy/Loki) before touching critical business data (Postgres).
- **Validation:** Each phase acts as a gate. The migration does not proceed to Postgres until k3s stability is confirmed.

### Negative/Trade-offs

- **Temporary Complexity:** The system will temporarily run a "Hybrid" stack (some services in Docker, some in k3s) requiring split-brain network configuration (e.g., exposing Docker ports to K3s).
- **Duration:** The migration will take longer than a "Big Bang" cutover.

## Verification

- [x] **Phase 1 Complete:** Alloy running in k3s, logs appearing in Docker-Loki.
- [x] **Phase 2 Complete:** Loki running in k3s, accepting logs from Alloy.
- [x] **Phase 3 Complete:** Grafana running in k3s, visualizing k3s-Loki.
- [ ] **Phase 4 Complete:** Postgres running in k3s, application traffic switched over.
