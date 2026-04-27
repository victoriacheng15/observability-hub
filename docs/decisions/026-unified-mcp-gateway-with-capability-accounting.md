# ADR 026: Unified MCP Gateway With Capability Accounting

- **Status:** Accepted
- **Date:** 2026-04-27
- **Author:** Victoria Cheng

## Context and Problem Statement

ADR 018 selected domain-isolated MCP binaries to reduce blast radius between telemetry, Kubernetes, and remediation capabilities. The implementation has since moved toward a unified MCP gateway because the active surface area is now limited to telemetry, pods, and network flows, while stale host and platform tools have been removed.

The old decision no longer matched the running architecture. The gateway also used a static startup message for tool count, which made soft-failed providers hard to diagnose when Kubernetes configuration or telemetry environment variables were missing.

## Decision Outcome

Adopt a unified MCP gateway binary for the active operational domains, with provider-level isolation and runtime capability accounting as the compensating controls.

- **Unified Binary:** `cmd/mcp-obs-hub` remains the single entry point for telemetry, pods, network, and diagnostic MCP tools.
- **Provider Boundaries:** Domain behavior stays isolated behind `internal/mcp/providers` and `internal/mcp/tools` packages.
- **Soft-Fail Visibility:** Startup records each domain as available or skipped, including skipped reasons.
- **Diagnostic Capability:** The gateway exposes `mcp_capabilities` so agents can inspect active tools and missing providers before attempting a workflow.
- **Reduced Blast Radius:** Host/platform MCP tools remain removed from the active gateway surface.

This ADR supersedes ADR 018 for the current MCP gateway deployment model. The split-binary approach remains a valid future option if host-level tools or materially broader permissions return.

## Consequences

### Positive

- **Simpler Operations:** Agents and local runtimes configure one MCP server instead of several domain-specific binaries.
- **Clearer Failures:** Provider startup failures are visible in logs and through `mcp_capabilities`.
- **Maintained Boundaries:** Provider and tool package boundaries preserve logical isolation inside the unified process.
- **Lower Tool Noise:** Removed host/platform tools keep the active gateway focused on telemetry, pods, and network flows.

### Negative

- **Shared Process Risk:** A single binary still shares one process boundary across multiple domains.
- **RBAC Discipline Required:** Kubernetes permissions must remain scoped to the active pod and network workflows.
- **Future Review Needed:** Reintroducing host-level actions should trigger a new ADR or a return to domain-isolated binaries.

## Verification

- [x] **ADR Reconciliation:** ADR 026 documents the unified gateway decision and supersedes ADR 018.
- [x] **Capability Accounting:** Startup records available and skipped MCP domains with actual tool counts.
- [x] **Diagnostic Tool:** `mcp_capabilities` reports the same capability state exposed in startup logs.
- [x] **Automated Tests:** `go test ./internal/mcp/...` passes.
