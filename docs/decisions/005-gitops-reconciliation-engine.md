# RFC 005: Centralized GitOps Reconciliation Engine

- **Status:** Accepted
- **Date:** 2026-01-03
- **Author:** Victoria Cheng

## The Problem

The primary bottleneck is the manual overhead and state drift that occurs when the repository is updated outside of the local terminal (e.g., merging a Pull Request via the GitHub Web UI).

While merging via `gh pr merge -s -d` handles local synchronization, web-based merges leave the server's local repository behind. This requires manual intervention (`git pull origin main`) to sync the "live" state with the "git" state, leading to unnecessary manual commands and potential human error in keeping services up to date.

## Validation & Technical Spike

Before committing to a full GitOps engine, the Systemd Timer approach was validated through the implementation of the `reading-sync` and `system-metrics` services. This spike confirmed the suitability of Linux primitives for this use case by verifying:

- **Journald Integration**: Native log capture and rotation.
- **Reliability**: Use of the `Persistent=true` flag to handle catch-up during downtime.
- **Maintainability**: Deployment via generic Makefile targets for system-wide service management.
- **Unified Scheduling**: Benchmarking systemd's ability to manage diverse telemetry collectors (API polling vs. system probing) consistently.

## Implementation Roadmap

To maintain simplicity and ensure stability, the rollout follows a structured three-phase roadmap:

- **Phase 1: Observability Hub Sync:** Initial rollout focusing exclusively on automating `git pull` for the `observability-hub` repository. This validates the Systemd/Bash mechanism and eliminates the manual overhead for the core platform. (Updated: 2026-01-12)
  - **State Enforcement:** Automatically validates and switches to the `main` branch to prevent drift.
  - **Hygiene:** Implements automated cleanup of local branches (pruning) to maintain host cleanliness.
  - **Safety Barriers:** Aborts operation if uncommitted changes are detected to prevent data loss.
- **Phase 2: Project Expansion:** Expand the automated synchronization mechanism to other repositories within the homelab environment using Systemd templates.
- **Phase 3: State Enforcement (Docker):** Once the synchronization mechanism is stable across all repositories, the agent will be expanded to trigger service restarts (e.g., `docker compose up -d`) to apply configuration changes automatically.

## Solution Architecture

We implemented a "Pull-based" synchronization agent managed by **Templated Systemd Timers**. This approach prioritizes security, scalability, and observability.

### 1. The Controller (Bash Agent)

The core logic is contained in [scripts/gitops_sync.sh](../../scripts/gitops_sync.sh). The agent uses an **Allowlist** pattern to prevent unauthorized access and **Logfmt** (structured logging) for native Loki integration.

### 2. Scalability (Systemd Templates)

We use the `@` symbol to create a single template that can service multiple repositories. This directly enables **Phase 2** of the roadmap.

- **Unit:** `gitops-sync@.service`
- **Timer:** `gitops-sync@.timer`

To scale to a new repository (e.g., `mehub`), we simply enable a new instance without creating new service files:
`systemctl enable --now gitops-sync@mehub.timer`

## Comparison / Alternatives Considered

- **ArgoCD / Flux:** While powerful, these tools introduce significant resource overhead (RAM/CPU) for a single-node homelab. A Bash-based agent provides $O(1)$ overhead.
- **Standard Cron:** Functional but limited. We explicitly chose Systemd Timers to leverage native dependency management (`After=network.target`) and unified journald logging.

## Failure Modes (Operational Excellence)

- **Git Pull Conflict:** If the local state deviates manually, the sync will fail. **Mitigation:** The script will exit with a non-zero code, visible in `systemctl status`.
- **Unauthorized Access:** If a user tries to sync an arbitrary directory. **Mitigation:** The allowlist logic prevents the script from acting on any directory not explicitly approved.

## Conclusion

This "Hub-and-Spoke" GitOps strategy provides a low-overhead, secure automation pipeline. By using native Linux primitives and templating, we ensure the system stays in sync with the remote repository reliably, while the "Allowlist" mechanism adheres to Zero Trust principles.
