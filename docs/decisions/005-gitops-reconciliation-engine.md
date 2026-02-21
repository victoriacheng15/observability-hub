# ADR 005: Centralized GitOps Reconciliation Engine

- **Status:** Superseded by [ADR 008](./008-gitops-via-proxy-webhook.md)
- **Date:** 2026-01-03
- **Author:** Victoria Cheng

## Context and Problem Statement

The primary bottleneck is the manual overhead and state drift that occurs when the repository is updated outside of the local terminal (e.g., merging a Pull Request via the GitHub Web UI).

While merging via `gh pr merge -s -d` handles local synchronization, web-based merges leave the server's local repository behind. This requires manual intervention (`git pull origin main`) to sync the "live" state with the "git" state, leading to unnecessary manual commands and potential human error in keeping services up to date.

## Decision Outcome

We implemented a **"Pull-based" synchronization agent managed by Templated Systemd Timers**. This approach prioritizes security, scalability, and observability.

### The Controller (Bash Agent)

The core logic is contained in [scripts/gitops_sync.sh](../../scripts/gitops_sync.sh). The agent uses an **Allowlist** pattern to prevent unauthorized access and **Logfmt** (structured logging) for native Loki integration.

### Scalability (Systemd Templates)

We use the `@` symbol to create a single template that can service multiple repositories.

- **Unit:** `gitops-sync@.service`
- **Timer:** `gitops-sync@.timer`

## Consequences

### Positive

- **Efficiency:** $O(1)$ overhead compared to ArgoCD/Flux.
- **Integration:** Native journald logging and `After=network.target` dependency management.
- **Security:** "Allowlist" logic adheres to Zero Trust principles.

### Negative

- **Conflict Risk:** If local state deviates manually, the sync will fail (mitigated by non-zero exit codes).

## Verification

- [x] **Manual Check:** Verify `systemctl status gitops-sync@observability-hub` shows success.
- [x] **Automated Tests:** Validated via the implementation of `reading-sync` and `system-metrics` services.
