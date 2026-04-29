# ADR 027: Hub CLI Developer Experience

- **Status:** Accepted
- **Date:** 2026-04-28
- **Author:** Victoria Cheng

## Context and Problem Statement

Observability Hub's active runtime surface is Kubernetes-backed: services map
to pods, workloads, namespaces, logs, events, and health signals. A Hub
internal developer platform CLI gives that surface a service-first interface
for understanding runtime state, ownership, and service status.

Without a shared command surface, the same investigation path can require
switching between pod lists, workload status, service logs, cluster events, and
ad hoc Kubernetes commands. The CLI should give those workflows a consistent
vocabulary without hiding the underlying platform model.

The first implementation should establish the command shape, with
service-focused commands that cover Kubernetes service discovery, health
summaries, logs, events, and developer helper commands.

## Decision Outcome

Add a Hub internal developer platform CLI with the source entry point at
`cmd/idp` and internal logic under `internal/idp`.

- **CLI Entry Point:** Build the CLI as the repository's developer interface.
- **Service-First UX:** Design commands around services before pods or raw
  Kubernetes objects.
- **Workflow Vocabulary:** Use command groups that match recurring platform
  workflows: service inspection, cluster state, Kubernetes catalog validation,
  and environment discovery.
- **Read-Only First:** Start with command routing and help text before adding
  Kubernetes access.

## Consequences

### Positive

- **Fast Feedback:** The CLI shape can be tested as part of normal
  development.
- **Simple Operations:** A single CLI keeps the developer workflow direct.
- **Project Alignment:** The CLI follows the repository's `cmd/` and
  `internal/` boundaries.
- **Clear Growth Path:** Catalog, Kubernetes, telemetry, and helper behavior
  have an explicit command surface to build on.

### Negative

- **Implementation Scope:** The first version proves command routing before
  deeper Kubernetes, telemetry, or remediation behavior.
- **Command Names Become Sticky:** Once commands such as `service health` or
  `cluster status` are documented, changing them later can break habits,
  scripts, or notes.

## Verification

- [x] **Service Commands:**
  - [x] `service list` is routed.
  - [x] `service describe <service>` is routed.
  - [x] `service health <service>` is routed.
  - [x] `service logs <service>` is routed.
  - [x] `service events <service>` is routed.
  - [x] `service metrics <service>` is routed.
  - [x] `service traces <service>` is routed.
  - [x] `service ownership <service>` is routed.
- [x] **Cluster Commands:**
  - [x] `cluster status` is routed.
  - [x] `cluster namespaces` is routed.
  - [x] `cluster workloads` is routed.
- [x] **Catalog Commands:**
  - [x] `catalog list` is routed.
  - [x] `catalog validate` is routed.
- [x] **Environment Commands:**
  - [x] `env list` is routed.
  - [x] `env describe <env>` is routed.
