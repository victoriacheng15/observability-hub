# ADR 027: Hub CLI Developer Experience

- **Status:** Accepted
- **Date:** 2026-04-28
- **Author:** Victoria Cheng

## Context and Problem Statement

Observability Hub has operational surfaces across Kubernetes, telemetry, MCP
tools, documentation, and host services. A local IDP CLI gives those surfaces a
service-first developer interface for understanding runtime state, signals,
ownership, and next steps.

The first implementation should establish the command shape as a local Linux
binary, with service-focused commands that can grow into Kubernetes service
discovery, health summaries, logs, events, and developer helper commands.

## Decision Outcome

Add a local Hub IDP CLI with the source entry point at `cmd/idp` and internal
logic under `internal/idp`.

- **Local Binary:** Build the CLI as a local Linux binary for day-to-day
  development.
- **Service-First UX:** Design commands around services before pods or raw
  Kubernetes objects.
- **Read-Only First:** Start with command routing and help text before adding
  Kubernetes access.

## Consequences

### Positive

- **Fast Feedback:** The CLI shape can be tested locally as part of normal
  development.
- **Simple Operations:** A single Linux binary keeps the local workflow direct.
- **Project Alignment:** The CLI follows the repository's `cmd/` and
  `internal/` boundaries.
- **Clear Growth Path:** Catalog, Kubernetes, telemetry, and helper behavior
  have an explicit command surface to build on.

### Negative

- **Linux-First Assumption:** The initial binary targets the local Linux
  environment instead of proving cross-platform behavior.
- **Command Names Become Sticky:** Once commands such as `service health` or
  `cluster status` are documented, changing them later can break habits,
  scripts, or notes.

## Verification

- [ ] **Service Commands:**
  - [ ] `service list` is routed.
  - [ ] `service describe <service>` is routed.
  - [ ] `service health <service>` is routed.
  - [ ] `service logs <service>` is routed.
  - [ ] `service events <service>` is routed.
  - [ ] `service metrics <service>` is routed.
  - [ ] `service traces <service>` is routed.
  - [ ] `service ownership <service>` is routed.
- [ ] **Cluster Commands:**
  - [ ] `cluster status` is routed.
  - [ ] `cluster namespaces` is routed.
  - [ ] `cluster workloads` is routed.
- [ ] **Catalog Commands:**
  - [ ] `catalog list` is routed.
  - [ ] `catalog validate` is routed.
- [ ] **Environment Commands:**
  - [ ] `env list` is routed.
  - [ ] `env describe <env>` is routed.
