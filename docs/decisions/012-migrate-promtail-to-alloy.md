# ADR 012: Migrate Promtail to Alloy

- **Status:** Proposed
- **Date:** 2026-02-01
- **Author:** Victoria Cheng

## Context and Problem Statement

Currently, the observability stack relies on **Promtail** (running in Docker) to scrape logs. However, Promtail has been effectively superseded by Grafana Alloy and is entering maintenance mode/End-of-Life (EOL). Continuing to rely on it incurs technical debt and risks missing future security updates and features.

## Decision Outcome

Adopt **Grafana Alloy** as the unified telemetry collector, replacing the legacy Promtail agent.

### Implementation Strategy: "Strangler Fig"

The "Strangler Fig" pattern will be used to minimize risk:

1. **Phase 1 (Shadow):** Deploy Alloy as a k3s `DaemonSet`. It will be configured to scrape k3s pods *and* mount the host's Docker log directory (`/var/lib/docker/containers`). It will push logs to Loki alongside Promtail.
2. **Phase 2 (Verify):** Confirm in Grafana that Alloy is successfully ingesting logs from both sources (tagged appropriately).
3. **Phase 3 (Cutover):** Stop the Promtail container in Docker Compose. Remove Promtail configuration.

### Key Configuration Changes

- **Deployment Model:** k3s DaemonSet (ensures it runs on every node).
- **Log Discovery:**
  - **Kubernetes:** Native discovery via the Kubernetes API.
  - **Docker:** Static path scraping via host volume mounts.
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

- [ ] **Alloy Running:** `kubectl get pods -l app=alloy` shows healthy status.
- [ ] **Log Ingestion:** Querying `{agent="alloy"}` in Grafana returns logs from both systemd and docker sources.
- [ ] **Parity Check:** Ensure timestamps and labels match the format previously provided by Promtail.
