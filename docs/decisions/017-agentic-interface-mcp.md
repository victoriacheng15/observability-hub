# ADR 017: Agentic Interface via MCP

- **Status:** Proposed
- **Date:** 2026-03-05
- **Author:** Victoria Cheng

## Context and Problem Statement

As the Observability Hub matures, the volume of telemetry data (Logs, Metrics, Traces) has outpaced the efficiency of human-centric dashboarding. Resolving complex incidents requires an engineer to manually correlate data across multiple distinct interfaces (Grafana, Tempo, Prometheus).

Simultaneously, as the industry moves toward Agentic Infrastructure (where AI agents assist in root cause analysis and governance), this implementation serves as an opportunity to implement the Model Context Protocol (MCP) and explore the practical challenges of building an AI-native observability interface.

To bridge the gap between the high-fidelity OpenTelemetry foundation and these emerging workflows, the platform requires a standardized, machine-readable interface.

This decision builds upon foundational refactorings that established the structural and data predictability required for an agentic interface. Key impact-oriented preparations included:

- **Architectural Mapping:** Flattened the monorepo into `cmd/` and `internal/` packages, allowing agents to reliably locate and reuse service logic.
- **Machine-Readable State:** Migrated infrastructure to OpenTofu (ADR-016), replacing imperative scripts with a declarative graph that an agent can reason about.
- **High-Fidelity Telemetry:** Standardized observability signals via the "Pure Wrapper" pattern and unified host collectors (ADR-015), ensuring consistent, correlated data across the entire stack.
These changes transformed the repository from a collection of services into a cohesive, observable platform ready for protocol-driven automation.

## Decision Outcome

Adopt the Model Context Protocol (MCP) to implement a **dedicated telemetry control plane**:

- **MCP-Telemetry (The Health Brain):** A dedicated MCP server acting as the primary professional artifact. It will provide intent-based tools (`query_metrics`, `query_logs`, `query_traces`, `investigate_incident`) that abstract the underlying LGTM stack.

To enable low-latency, direct-to-pod communication for this host-based MCP server, the implementation standardizes on a **NodePort (`localhost`) bridging strategy** for the internal cluster monitoring services (Loki, Thanos, Tempo).

## Consequences

### Positive

- **Reduced MTTD/MTTR:** AI agents can perform multi-dimensional correlation across the "Three Pillars" in a single reasoning loop, drastically accelerating incident response.
- **Architectural Decoupling:** By using intent-based tool naming, the AI's reasoning logic remains insulated from the underlying storage implementation.
- **Improved Governance:** Providing a standardized protocol for telemetry access enables better auditing and control over how automated agents interact with system data.

### Negative

- **Query Resource Intensity:** Automated agents may execute computationally expensive telemetry queries that could impact the performance or stability of the internal monitoring services.
- **Context Window Exhaustion:** High-volume telemetry retrieval (e.g., large distributed traces) can exceed the AI agent's context limits, leading to incomplete or inaccurate analysis.

## Verification

- [ ] **Level 0 (Infrastructure):** Verified Loki, Thanos, and Tempo are accessible via NodePort on `localhost`.
- [ ] **Level 1 (Metrics Intelligence):** Verified `mcp-telemetry` can perform autonomous service health analysis and performance baselining.
- [ ] **Level 2 (Semantic Logging):** Verified `mcp-telemetry` can correlate unstructured events with system failures via semantic LogQL filtering.
- [ ] **Level 3 (Trace Correlation):** Verified `mcp-telemetry` can reason over distributed request paths and parent/child span relationships.
- [ ] **Level 4 (Autonomous Investigator):** Verified the `investigate_incident` macro-tool can generate a complete, verifiable markdown RCA report.
