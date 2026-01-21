# RFC 008: GitOps via Proxy Webhook

- **Status:** Proposed
- **Date:** 2026-01-21
- **Author:** Victoria Cheng

## The Problem

Currently, we leverage `systemd` timers (specifically the template unit `gitops-sync@.timer`) to periodically trigger GitOps synchronizations for our repositories. While effective for a small number of services, this approach is becoming an operational bottleneck as our portfolio grows.

**Key Pain Points:**

- **Management Overhead:** Forecasting growth to 10+ repositories, the burden of manually instantiating and enabling a new systemd timer unit for each (e.g., `systemctl enable --now gitops-sync@new-repo.timer`) becomes unsustainable.
- **Latency:** The current implementation polls every 15 minutes. This delay is suboptimal for development loops where we want faster feedback.
- **Process Sprawl:** We are running multiple independent timer processes that essentially do the same thing.

## Proposed Solution

We propose shifting the trigger mechanism from `systemd` timers to an event-driven model using our existing Go Proxy service. A single endpoint will dynamically handle multiple repositories.

**Implementation Details:**

- **New Endpoint:** Add a single `/api/webhook/gitops` endpoint to the Go Proxy to act as a universal receiver for all managed repositories.
- **GitHub Integration:** Configure GitHub Webhooks across all relevant repositories to point to this unified endpoint.
- **Dynamic Execution:** The Go Proxy will parse the repository name from the JSON payload and pass it as an argument to `scripts/gitops_sync.sh`. This removes the need for per-repo configuration in the proxy itself.
- **Security:** Implement HMAC SHA-256 signature verification using a shared `GITHUB_WEBHOOK_SECRET`.

**Workflow:**
`GitHub Push` -> `Webhook Event` -> `Go Proxy Handler` -> `Verify Signature` -> `Execute gitops_sync.sh <repo>`

## Comparison / Alternatives Considered

| Approach | Pros | Cons |
| :--- | :--- | :--- |
| **Systemd Timers (Current)** | robust, native linux, independent of network | high management overhead, high latency (polling), requires ssh access to config |
| **Go Proxy Webhook (Proposed)** | real-time sync, single configuration point, leverages existing proxy | introduces dependency on proxy availability, requires exposing endpoint |

## Failure Modes (Operational Excellence)

- **Concurrency Issues:** Rapid pushes could trigger multiple concurrent sync scripts for the same repo, potentially causing git lock contention.
  - *Mitigation:* Implement a mutex or worker pool in the Go handler to serialize syncs per repository.
- **Proxy Downtime:** If the proxy crashes, syncs stop.
  - *Mitigation:* `systemd` handles proxy restarts. We could keep a fallback "catch-all" daily systemd timer just in case, or rely on manual syncs for recovery.

## Conclusion

Moving to a webhook-based approach aligns with our goal of reducing operational toil and improving system responsiveness. The Go Proxy is already a central component of our architecture, making it the logical place to host this logic. We will retain the `gitops_sync.sh` script as the execution engine, keeping the logic decoupled from the trigger mechanism.
