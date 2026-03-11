# ADR 018: Domain-Isolated MCP Architecture

- **Status:** Accepted
- **Date:** 2026-03-11
- **Author:** Victoria Cheng

## Context and Problem Statement

Following the adoption of the Model Context Protocol (ADR 017), the initial implementation focused exclusively on telemetry data (Thanos, Loki, Tempo). However, as the "Agentic Expansion" phase of the roadmap (Phase 2) begins, there is a requirement to expose infrastructure-level state—specifically Kubernetes Pods and Events—to the AI control plane.

Combining infrastructure tools (Kubernetes API) with telemetry tools (LGTM stack) into a single MCP server creates several architectural risks:

1. **Security Over-Privilege:** A monolithic server would require both network access to telemetry NodePorts and high-level RBAC permissions for the Kubernetes API, increasing the blast radius of a process compromise.
2. **Dependency Bloat:** The server would need to import both `client-go` and various telemetry clients, leading to larger binaries and slower startup times.
3. **Operational Fragility:** A crash or timeout in the Kubernetes client could disrupt the availability of the telemetry tools, breaking the agent's entire reasoning loop.

## Decision Outcome

Adopt a **Domain-Isolated MCP Architecture** by splitting agentic capabilities into specialized, standalone binaries:

- **Specialized Servers:** Implement a new dedicated service, **`mcp-pods`**, to handle all infrastructure-related operations and investigations. This includes diagnostic log retrieval (`get_pod_logs`) and basic remediation (`delete_pod`) alongside resource inspection.
- **Shared Registry Pattern:** Use a modular registry in `internal/mcp/registry.go` to share protocol logic (JSON-RPC formatting, error handling) while allowing each binary to register only the toolsets it requires.
- **Standalone Binary Pattern:** Deploy these servers as pure binaries communicating over `stdio`, avoiding host-tier service managers like systemd to improve portability and alignment with future containerized orchestration.
- **Lifecycle Standardization:** Enforce a unified signal-handling pattern (`os/signal`) across all MCP binaries to ensure reliable telemetry flushing and resource cleanup (e.g., closing provider connections) during server termination.

## Consequences

### Positive

- **Granular Security:** RBAC and network permissions can be applied per-service, following the Principle of Least Privilege.
- **Improved Maintainability:** Developers can work on infrastructure tools (`mcp-pods`) without risk of regressing telemetry logic.
- **Optimized Context:** AI agents can connect only to the servers they need for a specific task, reducing context window noise and protocol overhead.

### Negative

- **Configuration Complexity:** Users must manage multiple MCP server configurations in their CLI settings.
- **Binary Proliferation:** Increasing the number of binaries in the `bin/` directory and targets in the `Makefile`.

## Verification

- [x] **Modular Registry:** Refactored `internal/mcp/registry.go` to support `RegisterTelemetryTools` and `RegisterPodsTools` independently.
- [x] **Service Isolation:** Verified `mcp-pods` binary successfully executes and connects to the K3s API using `client-go` with pluralized naming conventions.
- [x] **Operational Expansion:** Implemented and verified `get_pod_logs` and `delete_pod` for active troubleshooting and remediation.
- [x] **Lifecycle Standardization:** Standardized all MCP servers (pods, telemetry) on signal-based graceful shutdown patterns to ensure reliable cleanup.
- [x] **Systemic Stability:** Confirmed that `mcp-telemetry` remains fully functional and isolated from the infrastructure-level logic.
