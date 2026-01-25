# ADR 008: GitOps via Proxy Webhook

- **Status:** Accepted
- **Date:** 2026-01-21
- **Author:** Victoria Cheng

## Context and Problem Statement

Currently, we leverage `systemd` timers to periodically trigger GitOps synchronizations. While effective for a small number of services, this approach is becoming an operational bottleneck due to management overhead, latency (polling), and process sprawl.

## Decision Outcome

Shift the trigger mechanism to an **Event-Driven Model** using the existing Go Proxy service.

- **New Endpoint:** `/api/webhook/gitops` acts as a universal receiver.
- **Dynamic Execution:** Parses repo name from payload and executes `gitops_sync.sh`.
- **Security:** HMAC SHA-256 signature verification.

## Consequences

### Positive

- **Real-time:** Updates happen immediately on push (vs. 15min polling).
- **Simplicity:** Single configuration point (Github Webhook) vs. multiple systemd timers.
- **Toil Reduction:** No need to SSH and create new timer units for new repos.

### Negative/Trade-offs

- **Dependency:** Relies on Proxy availability (unlike systemd timers which are independent).

## Verification

- [x] **Manual Check:** Push to a repo and observe log output in Proxy.
- [x] **Automated Tests:** Verify webhook signature validation logic in Proxy.
