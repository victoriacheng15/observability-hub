# RFC 005: Centralized GitOps Reconciliation Engine

**Status:** Proposed
**Date:** 2026-01-03
**Author:** Victoria Cheng

## The Problem

The primary bottleneck is the manual overhead and state drift that occurs when the repository is updated outside of the local terminal (e.g., merging a Pull Request via the GitHub Web UI).

While merging via `gh pr merge -s -d` handles local synchronization, web-based merges leave the server's local repository behind. This requires manual intervention (`git pull origin main`) to sync the "live" state with the "git" state, leading to unnecessary manual commands and potential human error in keeping services up to date.

## Validation & Technical Spike

Before committing to a full GitOps engine, the Systemd Timer approach was validated through the implementation of the `reading-sync` service. This spike confirmed the suitability of Linux primitives for this use case by verifying:

- **Journald Integration**: Native log capture and rotation.
- **Reliability**: Use of the `Persistent=true` flag to handle catch-up during downtime.
- **Maintainability**: Deployment via generic Makefile targets for system-wide service management.

## Proposed Solution

Implement a "Pull-based" synchronization agent managed by **Systemd Timers**. To maintain simplicity and ensure stability, the rollout follows a structured three-phase roadmap:

- **Phase 1: Observability Hub Sync:** Initial rollout focusing exclusively on automating `git pull` for the `observability-hub` repository. This validates the Systemd/Bash mechanism and eliminates the manual overhead for the core platform.
- **Phase 2: Project Expansion:** Expand the automated synchronization mechanism to other repositories within the homelab environment.
- **Phase 3: State Enforcement (Docker):** Once the synchronization mechanism is stable across all repositories, the agent will be expanded to trigger service restarts (e.g., `docker compose up -d`) to apply configuration changes automatically.

- **Systemd Integration:** Using `Type=oneshot` services and `Persistent=true` timers ensures that synchronization is reliable, dependency-aware (starts after network/docker), and natively integrated with `journald` for observability.

### Architecture Snippet (The Controller)

```bash
#!/bin/bash
# gitops-sync: A repository synchronization agent
set -euo pipefail

REPO_PATH=$1
cd "$REPO_PATH"

git fetch origin main --quiet
LOCAL_HASH=$(git rev-parse HEAD)
REMOTE_HASH=$(git rev-parse origin/main)

if [ "$LOCAL_HASH" != "$REMOTE_HASH" ]; then
    echo "Changes detected. Synchronizing repository..."
    git pull origin main
    # TODO: Trigger container updates in Phase 3
else
    echo "Repository in sync."
fi
```

## Comparison / Alternatives Considered

- **ArgoCD / Flux:** While powerful, these tools introduce significant resource overhead (RAM/CPU) for a single-node homelab. A Bash-based agent provides $O(1)$ overhead.
- **Standard Cron:** Functional but limited. While Cron is the standard for scheduled tasks, we are explicitly choosing Systemd Timers to **evaluate** their suitability for infrastructure orchestration. This decision is driven by a desire to benchmark Systemd's native dependency management and logging capabilities against the traditional Cron approach.

## Failure Modes (Operational Excellence)

- **Git Pull Conflict:** If the local state deviates manually, the sync will fail. **Mitigation:** The script will exit with a non-zero code, visible in `systemctl status`, and can be configured to send an alert to the Observability Hub.
- **Runtime Downtime:** If Docker is down during a sync, docker commands will fail. **Mitigation:** Systemd's `After=docker.service` ensures the sync only runs when the cluster is healthy.

## Conclusion

This "Hub-and-Spoke" GitOps strategy provides a low-overhead automation pipeline that eliminates manual synchronization tasks. By using native Linux primitives, we ensure the system stays in sync with the remote repository reliably and without manual `git pull` commands.
