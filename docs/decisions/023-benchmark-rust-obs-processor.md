# ADR 023: Benchmark-Validated Rust Obs Processor

- **Status:** Accepted
- **Date:** 2026-04-17
- **Author:** Victoria Cheng

## Context and Problem Statement

ADR 021 selected Rust for the `obs-processor` sidecar that summarizes logs and metrics before telemetry responses are returned to MCP agents. ADR 022 refined the summary format so the processor preserves investigation pivots while still reducing raw payload size.

The remaining question is whether Rust should stay as the long-term implementation choice for this processing path instead of moving the summarization logic back into Go. The platform is Go-first for service orchestration, but log and metric reduction has a different performance profile: it reads large JSON payloads, groups repeated log lines, computes metric statistics, and reduces output before the data becomes agent context.

This matters because raw Loki and Prometheus responses can become expensive in two ways:

- **Runtime cost:** Large payloads require CPU and memory to parse, group, and summarize.
- **Token cost:** Large responses consume context window budget and increase the cost of agent-assisted investigations.
- **Operator cost:** Noisy responses make it harder to identify the signal that actually matters.

The decision needs benchmark evidence, not only a language preference.

## Decision Outcome

Keep the production telemetry reducer in Rust and maintain a dedicated benchmark harness that compares equivalent Go and Rust processors against the same live API payload.

- **Benchmark Harness:** `scripts/benchmark_obs_processor.sh` fetches either a Loki or Prometheus payload, builds both standalone processors, runs repeated iterations, and reports average, minimum, and maximum runtime.
- **Go Baseline:** `scripts/bench/obs_processor.go` provides a comparable Go implementation of the same summarization behavior.
- **Rust Candidate:** `scripts/bench/obs_processor.rs` provides the Rust implementation used to validate the performance characteristics of the sidecar approach.
- **Payload Parity:** Both binaries process the same captured payload from stdin, which keeps the comparison focused on parsing and summarization rather than network timing.
- **Token-Cost Link:** Runtime benchmarks validate processor efficiency, while the ROI benchmark from ADR 021 and ADR 022 validates byte, token, and cost reduction from raw telemetry to summaries.

### Rationale

- **Rust is appropriate for the hot path:** The processor performs bounded, CPU-sensitive transformation over large structured telemetry payloads.
- **Go remains the control plane:** MCP handlers still own provider calls, validation, timeouts, and fail-open behavior.
- **Benchmarks prevent hand-waving:** The language decision is tied to repeatable measurements under `scripts/bench` rather than preference.
- **Token reduction is the product outcome:** Faster parsing is useful, but the operational value is reducing context waste while preserving enough diagnostic structure for investigations.
- **The boundary stays narrow:** Rust is used for telemetry reduction only, avoiding a broad polyglot rewrite of platform services.

## Consequences

### Positive

- **Measured implementation choice:** Rust remains justified by a benchmarkable processing workload.
- **Lower agent context waste:** Logs and metrics are summarized before they become expensive prompt context.
- **Clear regression guard:** Future changes can rerun the benchmark to catch slower parsing or less efficient summaries.
- **Separation of concerns:** Go handles orchestration and safety; Rust handles high-volume payload transformation.

### Negative

- **Additional benchmark maintenance:** Go and Rust benchmark implementations must stay behaviorally comparable.
- **Toolchain complexity:** Benchmarking requires both Go and Cargo to be available.
- **Representative payload dependency:** Benchmark quality depends on querying realistic Loki and Prometheus data.

## Verification

- [x] **Benchmark Script:** `scripts/benchmark_obs_processor.sh` added for Go vs Rust processor comparison.
- [x] **Go Benchmark Processor:** `scripts/bench/obs_processor.go` added as the Go baseline.
- [x] **Rust Benchmark Processor:** `scripts/bench/obs_processor.rs` added as the Rust comparison target.
- [x] **ROI Linkage:** ADR 021 and ADR 022 continue to document raw-to-summary byte, token, and cost reduction.

## Benchmark Result

Representative 24-hour logs and metrics benchmarks used the same API payload for both processors and ran 20 iterations per telemetry type.

### Logs

| Processor | Average | Minimum | Maximum | Iterations |
| :--- | ---: | ---: | ---: | ---: |
| Go | `4.782ms` | `4.108ms` | `6.316ms` | `20` |
| Rust | `2.608ms` | `2.125ms` | `3.029ms` | `20` |

Rust completed the same log-processing workload about `1.83x` faster on average for this benchmark run.

### Metrics

| Processor | Average | Minimum | Maximum | Iterations |
| :--- | ---: | ---: | ---: | ---: |
| Go | `18.460ms` | `16.665ms` | `20.062ms` | `20` |
| Rust | `4.943ms` | `4.328ms` | `6.224ms` | `20` |

Rust completed the same metric-processing workload about `3.73x` faster on average for this benchmark run.
