# ADR 025: Decouple Hardware Simulation

- **Status:** Accepted
- **Date:** 2026-04-25
- **Author:** Victoria Cheng

## Context and Problem Statement

`observability-hub` had been carrying two different responsibilities in the
same repo:

- platform and observability ownership
- hardware simulation workloads used as a telemetry-producing learning lab

That arrangement was useful while the simulation was being developed alongside
the platform, but it created an increasingly blurry boundary. The repo was
mixing platform concerns such as Argo CD bootstrap, shared observability
infrastructure, and operator tooling with workload-specific concerns such as the
sensor fleet, chaos controller, MQTT ingestor, hardware Dockerfiles, and
hardware Kubernetes manifests.

The result was avoidable coupling:

- platform CI/CD still built hardware images
- the active `k3s/` tree rendered hardware workloads directly
- architecture and workflow docs implied hardware was still an internal service
  domain
- the repo drifted away from a pure platform and observability focus

The project needs a cleaner platform boundary: `observability-hub` should own
the shared control plane, telemetry plane, and operator workflows, while the
hardware simulation should evolve independently as an external workload repo.

## Decision Outcome

Extract the hardware simulation domain into a separate public repo,
`hardware-sim-lab`, and keep `observability-hub` focused on platform and
observability ownership.

- **Workload Ownership Moves Out:** `sensor`, `chaos-controller`,
  `mqtt-ingestor`, their Dockerfiles, and their Kubernetes manifests move to
  `hardware-sim-lab`.
- **Telemetry Contract Stays Shared:** The hardware repo continues to emit to
  the shared MQTT and OpenTelemetry environment so Grafana and Prometheus views
  remain part of the same operational plane.
- **Private Internals Stay Private:** The new repo gets its own
  `internal/telemetry` and related internal packages instead of importing
  `observability-hub/internal/...` across repo boundaries.
- **GitOps Control Stays Here:** `observability-hub` keeps Argo CD bootstrap and
  owns child `Application` objects that point at `hardware-sim-lab` overlays.
- **Active Kustomize Ownership Ends Here:** The old in-repo hardware base,
  overlays, and hardware-only RBAC are removed from the active `k3s/` tree in
  `observability-hub`.

### Rationale

- **Platform boundary becomes explicit:** This repo returns to its primary role
  as the platform, observability, and operator control plane.
- **Hardware can evolve independently:** The simulation can ship code, images,
  and manifests without dragging unrelated platform changes into the same review
  path.
- **GitOps ownership becomes cleaner:** Argo CD app-of-apps is easier to reason
  about than a mixed repo where root manifests and extracted workloads overlap.
- **Interfaces become real contracts:** MQTT topics, payload shape, and OTEL
  conventions become external integration points instead of accidental in-repo
  coupling.
- **Documentation aligns with reality:** Architecture and workflow docs can
  describe the platform as observing external workloads rather than embedding
  them.

## Consequences

### Positive

- **Cleaner repo purpose:** `observability-hub` stays centered on platform and
  observability concerns.
- **Independent release cadence:** `hardware-sim-lab` can build and publish its
  own images without coupling to platform image workflows.
- **Clearer GitOps control plane:** Argo CD child apps in this repo point to the
  new workload repo instead of reconciling the hardware manifests directly.
- **Better long-term maintainability:** Internal package boundaries, docs, and
  CI ownership are easier to keep coherent.

### Negative

- **More moving parts:** Two repos must stay aligned on MQTT, telemetry labels,
  and deployment expectations.
- **Cutover requires a clean handoff:** Argo CD ownership needs a one-time
  transition from the old in-repo manifests to the child applications in order
  to avoid duplicate resource management.
- **Docs split across repos:** Some operational knowledge now belongs in the
  hardware repo while platform-facing guidance stays here.

## Verification

- [x] **Code Ownership Cleanup:** Hardware build/source ownership was removed
  from `observability-hub`.
- [x] **Kustomize Render:** `kubectl kustomize k3s` completed successfully after
  removing the old in-repo hardware manifests.
- [x] **Argo CD Structure:** Child `Application` manifests for
  `hardware-sim-lab-dev` and `hardware-sim-lab-prod` were added under
  `k3s/base/hub-apps`.
- [x] **Documentation:** Core platform docs were updated to describe hardware
  simulation as an external workload repo.
