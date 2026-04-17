# ADR 008: GitOps via Proxy Webhook

- **Status:** Accepted
- **Date:** 2026-01-21
- **Author:** Victoria Cheng

## Context and Problem Statement

The platform previously used `systemd` timers to periodically trigger GitOps synchronizations. While simple, timer-based polling caused the machine to wake up and check for work even when no repository had changed. That behavior wasted CPU cycles and power, added avoidable background activity, and made synchronization depend on a fixed schedule instead of actual change events.

Polling also weakened the source-of-truth loop: Git changes could sit unapplied until the next timer interval, and each additional synchronized service required another timer surface to manage, monitor, and debug.

The webhook path makes reconciliation event-driven. When GitHub sends a webhook, the Proxy receives the event, authenticates it, maps it to an allowlisted repository, executes the sync script, and emits structured logs for the result. If no Git event occurs, no sync work is triggered.

## Decision Outcome

Shift the trigger mechanism to an **Event-Driven Model** using the existing Go Proxy service.

- **New Endpoint:** `/api/webhook/gitops` acts as a universal receiver.
- **Dynamic Execution:** Parses repo name from payload and executes `gitops_sync.sh`.
- **Security:** HMAC SHA-256 signature verification.
- **Observability:** Proxy logs provide one place to trace webhook receipt, validation, and sync execution.
- **Bounded Execution:** Repository names are resolved through the existing sync script controls instead of accepting arbitrary shell input.

## Consequences

### Positive

- **Real-time:** Updates happen immediately on push (vs. 15min polling).
- **Simplicity:** Single configuration point (GitHub Webhook) vs. multiple systemd timers.
- **Toil Reduction:** No need to SSH and create new timer units for new repos.
- **Auditability:** Each sync attempt has a request path, validation result, and execution log.

### Negative

- **Dependency:** Relies on Proxy availability (unlike systemd timers which are independent).
- **Blast Radius:** A malformed webhook handler could affect multiple synchronization paths, so signature checks and allowlisted repo execution are required.

## Verification

- [x] **Manual Check:** Push to a repo and observe log output in Proxy.
- [x] **Automated Tests:** Verify webhook signature validation logic in Proxy.
