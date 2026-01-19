# RFC 007: k3s Shadow Deployment & Orchestration

- **Status:** Proposed
- **Date:** 2026-01-19
- **Author:** Victoria Cheng

## The Problem

While Docker Compose remains a functional and reliable foundation for the current repository, there is a desire to explore the potential of k3s and industry-standard orchestration.

Relying exclusively on Docker Compose limits the opportunity to evaluate:

1. **Orchestration Patterns:** Native self-healing, rolling updates, and the Gateway API.
2. **Platform Resiliency:** Moving toward a more robust reconciliation loop.
3. **Architectural Scaling:** Understanding the trade-offs between simple container management and full-scale orchestration.

## Proposed Solution

Implement a **Shadow Deployment Strategy**. This involves running an experimental "v2" version of services in k3s alongside the production version in Docker.

### Key Implementation Details

1. **Isolation:** Use `NodePort` on high-range ports (e.g., `30080`) to prevent production conflicts.
2. **Feature Toggling:** Use `SKIP_DB_INIT` environment variables to allow k3s pods to boot and serve static/dummy endpoints without needing complex database networking in the spike phase.
3. **Outbound Verification:** Implement external fetch endpoints (e.g., `/api/dummy`) to validate cluster DNS and internet egress.
4. **Local Image Injection:** Use `k3s ctr images import` to bridge the gap between Docker builds and the k3s containerd runtime without requiring an external registry.

## Comparison / Alternatives Considered

- **Full Migration:** High risk of downtime. Networking between k3s and Docker is non-trivial and requires careful planning for database access.
- **k3d:** Excellent for local dev but k3s is better suited for a long-running "production-like" homelab on the host.

## Failure Modes (Operational Excellence)

- **Port Conflict:** If a NodePort is chosen that is already in use by a Docker container or host service. **Mitigation:** Use the `30000-32767` range and explicitly check availability.
- **Image Stale State:** If the user forgets to `import` the new image, k3s will run the old one. **Mitigation:** Always use `imagePullPolicy: Never` to force a failure if the local image is missing.
- **Memory Pressure:** k3s and Docker share host RAM. **Mitigation:** Enforce strict `resources.limits` on all k3s manifests.

## Conclusion

By adopting a Shadow Deployment model, we can safely prototype k3s orchestration features while maintaining the 100% uptime of the production Docker stack. Once the k3s environment is verified (Networking, Secrets, Persistence), a phased migration of individual services can occur.
