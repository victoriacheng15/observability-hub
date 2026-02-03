# ADR 012: Migrate Promtail to Alloy

- **Status:** Accepted
- **Date:** 2026-02-01
- **Author:** Victoria Cheng

## Context and Problem Statement

Currently, the observability stack relies on **Promtail** (running in Docker) to scrape logs. However, Promtail has been effectively superseded by Grafana Alloy and is entering maintenance mode/End-of-Life (EOL). Continuing to rely on it incurs technical debt and risks missing future security updates and features.

## Decision Outcome

Adopt **Grafana Alloy** as the unified telemetry collector, replacing the legacy Promtail agent.

### Implementation Strategy: "Strangler Fig"

The "Strangler Fig" pattern was used to migrate log sources incrementally:

1. **Phase 1 (Systemd):** Deploy Alloy as a k3s `DaemonSet`. It was configured to scrape the host's `systemd` journal and push logs to the existing Docker-Loki instance alongside Promtail. This proved the viability of the Alloy pipeline.
2. **Phase 2 (Verify):** Confirm in Grafana that Alloy is successfully ingesting `systemd` journal logs with full label parity.
3. **Phase 3 (Cutover - Planned):** The `promtail` container in Docker Compose can now be disabled, as Alloy has taken over its responsibilities for host-level services.

### Key Configuration Changes

- **Deployment Model:** k3s DaemonSet (ensures it runs on every node).
- **Log Discovery:**
  - **Systemd:** Scraping of the host's journal via a `hostPath` volume mount (`/var/log/journal`).
- **Config Language:** Shift from YAML (Promtail) to **Alloy Config** (HCL-based), enabling more programmable and modular pipelines.

## Consequences

### Positive

- **Unified Stack:** A single binary for logs, metrics, and traces.
- **Kubernetes Native:** First-class support for Pod discovery and metadata enrichment.
- **Future Proofing:** Alloy is the successor to Promtail/Agent, ensuring long-term support and feature updates.

### Negative/Trade-offs

- **Learning Curve:** Alloy's configuration syntax (River/HCL) is different from Promtail's YAML.
- **Migration Effort:** Requires rewriting log processing pipelines (relabeling rules) into the new format.

## Verification

- [x] **Alloy Running:** `kubectl get pods -l app.kubernetes.io/name=alloy` shows a healthy status.
- [x] **Log Ingestion:** Querying `{agent="alloy"}` in Loki returns logs from systemd sources.
- [x] **Parity Check:** Labels (`service`, `unit`, etc.) and timestamps match the format previously provided by Promtail for systemd logs.
