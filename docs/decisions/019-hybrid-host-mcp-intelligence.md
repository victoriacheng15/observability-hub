# ADR 019: Hybrid Host-MCP Intelligence Layer

- **Status:** Accepted
- **Date:** 2026-03-12
- **Author:** Victoria Cheng

## Context and Problem Statement

The Observability Hub operates in a hybrid environment where high-performance data ingestion pipelines (`ingestion`, `proxy`) and core security infrastructure (`openbao`) reside as systemd services on the physical host, while the telemetry and visualization stack (LGTM) is orchestrated via Kubernetes (K3s).

Existing Model Context Protocol (MCP) implementations, such as `mcp-pods`, provide deep visibility into the cluster but are blind to the host-resident "Pillar Services." This created an intelligence gap where AI agents could not correlate cluster-level telemetry with host-level system state or logs.

## Decision Outcome

Implement a dedicated MCP server, `mcp-hub`, to act as the authoritative "Host-Cluster Bridge."

### Rationale

- **Hybrid Visibility:** Enables agents to diagnose "Dark Matter" failures occurring outside the Kubernetes runtime by wrapping system-native tools (`systemctl`, `journalctl`, `free`, `df`).
- **Semantic Clarity:** Strict namespacing with the `hub_` prefix prevents tool name collisions and provides a clear domain boundary for the agent.
- **Testability & Reliability:** Use of a `CommandRunner` interface allows for 100% unit test coverage of state-parsing logic without side-effects.
- **Architectural Consistency:** Aligns with the `internalmcp` pattern for unified telemetry and signal handling across all MCP services.

## Consequences

### Positive

- **Executive Oversight:** Provides a unified summary (`hub_inspect_platform`) that combines K3s and Host health status.
- **Direct-Path Ingestion:** Enables autonomous log analysis via systemd journal queries, bypassing the telemetry pipeline for real-time investigation.
- **Resource Awareness:** Agents can now correlate host-level resource pressure (Memory/CPU) with pod failures.

### Negative

- **Privileged Access:** Requires host-level execution permissions for systemd and kubectl, increasing the security surface area.
- **Dependency Bloat:** Adds another service to the MCP fleet that requires maintenance and monitoring.

## Verification

- [x] **Unit Tests:** `internal/mcp/providers/hub_test.go` and `internal/mcp/tools/hub_test.go` verify parsing logic and MCP handlers.
- [x] **Registry Check:** Tools are correctly registered with the `hub_` prefix in the implementation.
- [x] **Binary Verification:** `make mcp-hub-build` successfully compiles the service.
