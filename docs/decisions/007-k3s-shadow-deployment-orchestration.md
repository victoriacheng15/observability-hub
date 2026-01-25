# ADR 007: k3s Shadow Deployment & Orchestration

- **Status:** Accepted
- **Date:** 2026-01-19
- **Author:** Victoria Cheng

## Context and Problem Statement

While Docker Compose remains a functional and reliable foundation, there is a desire to explore the potential of k3s and industry-standard orchestration. Relying exclusively on Docker Compose limits the opportunity to evaluate orchestration patterns, platform resiliency, and architectural scaling.

## Decision Outcome

Implement a **Shadow Deployment Strategy**. This involves running an experimental "v2" version of services in k3s alongside the production version in Docker.

### Key Implementation Details

- **Isolation:** Use `NodePort` on high-range ports (e.g., `30080`).
- **Feature Toggling:** Use `SKIP_DB_INIT` to allow k3s pods to boot without complex DB networking.
- **Local Image Injection:** Use `k3s ctr images import` to bridge Docker builds and k3s.
- **Bootstrap Verification (`/api/dummy`):** A temporary endpoint that fetches `https://api.github.com/zen` to prove k3s egress/DNS works. (To be removed later).

## Consequences

### Positive

- **Risk Mitigation:** Allows prototyping k3s without downtime for the main stack.
- **Learning:** Enables evaluation of orchestration patterns (Gateway API, Rolling Updates).

### Negative/Trade-offs

- **Complexity:** Managing two runtimes (Docker + k3s) on the same host.
- **Resources:** k3s and Docker share host RAM (mitigated by resource limits).

## Verification

- [x] **Manual Check:** Verify k3s pods are running via `kubectl get pods`.
- [x] **Automated Tests:** Outbound verification via `/api/dummy` endpoint.
