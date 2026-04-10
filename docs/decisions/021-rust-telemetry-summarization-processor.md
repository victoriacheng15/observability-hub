# ADR 021: Rust Telemetry Summarization Processor

- **Status:** Accepted
- **Date:** 2026-04-06
- **Author:** Victoria Cheng

## Context and Problem Statement

The MCP telemetry layer already exposes direct access to metrics and logs, but raw Prometheus and Loki responses are often too verbose for efficient agent reasoning. Large payloads increase latency, waste context window budget, and make it harder for agents to extract the most important operational signals quickly.

This problem becomes worse as the query date range expands. Large windows produce a wall of repetitive logs and dense metric series, increasing the chance that an AI agent focuses on noise, misses the real error, or hallucinates patterns that are not operationally meaningful.

The existing Go-based MCP server remains the right place for query validation, transport integration, and fail-safe behavior. However, the response-reduction step has different constraints:

- **High-Volume Payload Handling:** Metrics vectors and log streams can be large enough that lightweight post-processing becomes performance-sensitive.
- **Isolation of Parsing Logic:** Summarization is a narrow transformation concern that benefits from being decoupled from the main MCP server process.
- **Operational Safety:** If summarization fails, the platform must still return raw telemetry rather than break the investigation path.
- **Context Compression:** The platform needs a consistent way to turn repetitive raw telemetry into compact summaries that preserve signal while reducing token volume.

This created a need for a small, focused processing boundary that can summarize telemetry without bloating the Go service or weakening the fail-open design of the MCP tools.

## Decision Outcome

Adopt a standalone **Rust telemetry processor** (`obs-processor`) as a helper binary for MCP log and metric summarization.

### Architectural Shape

- **Go owns the control plane:** Query validation, provider calls, timeout control, and fail-open behavior stay in the Go MCP handlers.
- **Rust owns the reduction step:** The `obs-processor` binary reads raw JSON from stdin and emits a summarized JSON structure to stdout.
- **Logs are repetition-aware:** Repeated log messages are grouped and emitted with occurrence counts such as `(xN)` so agents can see dominant failure patterns without reading every duplicate line.
- **Metrics are statistically reduced:** Metric responses are compressed into summary values such as minimum, maximum, average, p95, and trend direction rather than returning the full raw series by default.
- **Fail-open by design:** If the Rust processor is unavailable, times out, or returns invalid output, the Go handlers return the raw provider response instead of failing the tool call.
- **Explicit runtime dependency:** The MCP build/install path now includes the Rust binary as part of the production toolchain.

### Rationale

- **Performance-oriented parsing:** Rust is a strong fit for predictable, low-overhead processing of large structured telemetry payloads.
- **Better agent focus:** Summarizing repeated log lines and metric ranges reduces the chance that agents spend attention on volume instead of the underlying fault.
- **Tight blast radius:** Keeping summarization in a separate binary isolates parsing failures from the main MCP server process.
- **Polyglot where it matters:** The project stays Go-first for service architecture, but uses Rust for a narrow, computationally focused responsibility.
- **Preserved operator trust:** The fail-open contract ensures observability access still works even when summarization does not.
- **Measured ROI:** A dedicated benchmark script, [bench_sidecar_roi.sh](/home/server2/software/observability-hub/scripts/bench_sidecar_roi.sh), documents token, byte, and estimated cost reduction across representative log and metric queries.

## Consequences

### Positive

- **Better agent usability:** Logs and metrics can be reduced into compact summaries that are easier for agents to reason over.
- **Lower context waste:** Repetition-aware log compression and statistical metric summaries reduce token volume on wide time-range queries.
- **Process isolation:** Parser or summarizer faults are contained to a child process boundary rather than crashing the Go server.
- **Clear separation of concerns:** Go remains responsible for orchestration and safety, while Rust handles payload reduction.
- **Extensible processing path:** Additional telemetry transformations can be added to the processor without overloading MCP handler code.

### Negative

- **Polyglot build complexity:** The MCP path now depends on both Go and Rust toolchains.
- **Deployment coupling:** Production installs must place `obs-processor` at the expected runtime path.
- **Cross-process overhead:** JSON marshaling and stdin/stdout handoff add complexity compared with an in-process library.
- **Debugging surface area:** Failures may now occur in either the Go caller or the Rust processor boundary.
- **Temporal detail loss:** The current compression model is intentionally aggressive and can hide useful timing detail for repeated errors until richer timestamp-aware summaries are added.
- **Still evolving:** Metric summaries currently focus on core statistics such as min, max, average, and p95; richer summaries such as p99 may be added later.

## Verification

- [x] **Log Summarization:** Verified MCP log queries can invoke `obs-processor --type logs` and return summarized results.
- [x] **Metric Summarization:** Verified MCP metrics queries can invoke `obs-processor --type metrics` and return summarized results.
- [x] **Fail-Open Behavior:** Verified handlers fall back to raw telemetry when summarization fails.
- [x] **Build Integration:** Verified the MCP build flow compiles and installs `obs-processor` alongside `mcp_obs_hub`.
- [x] **ROI Benchmarking:** Verified [bench_sidecar_roi.sh](/home/server2/software/observability-hub/scripts/bench_sidecar_roi.sh) can measure raw vs summarized bytes, token estimates, density gain, and estimated cost reduction for representative telemetry queries.

## Benchmark Note

A representative 24-hour `proxy` logs benchmark produced the following reduction:

| Measure | Before | After | Change |
| :--- | :--- | :--- | :--- |
| Bytes | `369,745` | `352` | `-369,393` |
| Estimated Tokens | `92,436` | `88` | `-92,348` |
| Estimated Cost (USD) | `$0.2310` | `$0.0002` | `-$0.2308` |
| Estimated Cost (CAD) | `$0.3234` | `$0.0003` | `-$0.3231` |

The benchmark helper script is [bench_sidecar_roi.sh](/home/server2/software/observability-hub/scripts/bench_sidecar_roi.sh). Use the wrapper flags `--logs` or `--metrics` to measure the before/after reduction for each telemetry type.
