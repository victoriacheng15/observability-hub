# ADR 022: Structured Summaries for Obs Processor

- **Status:** Accepted
- **Date:** 2026-04-14
- **Author:** Victoria Cheng

## Context and Problem Statement

ADR 021 introduced `obs-processor`, a Rust helper binary that compresses raw Loki and Prometheus responses before the MCP telemetry tools return them to agents. That decision remains valid: Go owns query validation, provider calls, timeout control, and fail-open behavior, while Rust owns the response reduction step.

Live use showed that the first summary format is too lossy for investigation. It reduces payload size aggressively, but important diagnostic pivots disappear:

- **Repeated log errors lose cause:** An error such as `webhook_sync_failed (x2)` does not preserve structured fields like `repo`, `error`, `output`, or the failure window.
- **Timestamps disappear:** Grouped log entries do not expose first and last timestamps, making it harder to pivot into raw logs and traces around the event.
- **Warnings and info logs are not differentiated enough:** Normal high-volume info logs need minimal counts, while warnings and errors need more diagnostic context.
- **Metric summaries need clearer semantics:** Large Prometheus responses can emit one summarized entry per series, and that is acceptable while compression remains strong, but each entry needs clearer structure and metric-type semantics.
- **Counters are summarized like gauges:** Cumulative metrics such as `*_total` can show raw upward trends that are technically true but operationally misleading.

The platform needs a second-stage refinement that preserves the sidecar boundary and token savings while making summaries useful as investigation starting points.

## Decision Outcome

Refine `obs-processor` to emit structured, investigation-aware summaries instead of string-only entries.

### Target Summary Examples

Current log summaries collapse repeated errors into one-line strings:

```json
{
  "total_raw_lines": 46,
  "summarized_count": 8,
  "entries": [
    "[ERROR] webhook_sync_failed (x2)",
    "[INFO] request_processed (x14)"
  ]
}
```

The target log summary keeps info compact while preserving error pivots:

```json
{
  "total_raw_lines": 46,
  "summarized_count": 8,
  "entries": [
    {
      "level": "error",
      "message": "webhook_sync_failed",
      "count": 2,
      "first_timestamp_ns": "1776127999310444870",
      "last_timestamp_ns": "1776128000493256341",
      "context": {
        "service_name": "proxy",
        "repo": "bioHub",
        "error": "exit status 1",
        "output_preview": "Repository 'bioHub' is not a valid git repository."
      }
    },
    {
      "level": "info",
      "message": "request_processed",
      "count": 14
    }
  ],
  "omitted_entries": 0
}
```

Current metric summaries can still emit long one-line entries per series:

```json
{
  "total_raw_lines": 128,
  "summarized_count": 128,
  "entries": [
    "node_cpu_seconds_total{cpu=\"0\", job=\"kubernetes-service-endpoints\", instance=\"node-exporter:9100\", cluster=\"homelab\", mode=\"idle\"} | stats: [min:693177.38, max:696579.51, avg:694879.38, p95:696464.90] trend: up (+3402.13)"
  ]
}
```

The target metric summary keeps structured labels, summarizes counters with counter-aware fields, and leaves future filtering decisions to operational feedback:

```json
{
  "result_type": "matrix",
  "total_raw_lines": 128,
  "summarized_count": 128,
  "entries": [
    {
      "metric": "node_cpu_seconds_total",
      "kind": "counter",
      "status": "normal",
      "labels": {
        "cluster": "homelab",
        "cpu": "0",
        "instance": "node-exporter:9100",
        "job": "kubernetes-service-endpoints",
        "mode": "idle"
      },
      "sample_count": 6,
      "first": 701759.78,
      "last": 702047.6,
      "delta": 287.82,
      "average_rate_per_second": 0.9594,
      "resets_detected": 0,
      "first_timestamp": 1776135942.0,
      "last_timestamp": 1776136242.0
    }
  ]
}
```

### Log Summary Policy

Log summaries remain repetition-aware, but the amount of detail depends on severity:

- **Errors:** Include count, first timestamp, last timestamp, and bounded diagnostic context.
- **Warnings:** Include count, timestamps, and limited context when useful.
- **Info:** Keep minimal by default, usually level, message, and count.

Diagnostic context should be extracted from structured Loki stream metadata through a small allowlist. Useful fields include `service_name`, `repo`, `error`, `status`, `path`, `ref`, `action`, and a bounded `output_preview`.

### Metric Summary Policy

Metric summaries should distinguish metric semantics without prematurely dropping labels:

- **Anomalies:** Include status, retained labels, timestamps, current or expected values, and short context.
- **Counters:** Use delta, average rate, reset count, sample count, and timestamps.
- **Gauges:** Use min, max, average, p95, p99, first value, last value, trend delta, sample count, and timestamps.
- **Labels:** Retain labels for now so live usage can show which labels are actually useful for investigation.

Label filtering and grouping are deferred. The benchmark shows strong compression even with labels retained, so filtering should be revisited only if live usage shows that labels hurt readability, privacy, or investigation quality.

### Benchmark Policy

Benchmarking remains required. The benchmark must continue to prove that summarization preserves the practical ROI of the sidecar:

- Raw bytes
- Summary bytes
- Estimated tokens
- Estimated savings
- Context-density gain

The existing benchmark script continued to work after structured summaries landed, so no benchmark script change was required.

Representative validation after the structured refactor:

| Type | Raw Bytes | Summary Bytes | Context Gain |
| :--- | ---: | ---: | ---: |
| Logs | `566,863` | `6,280` | `~90.2x` |
| Metrics | `4,169,708` | `51,747` | `~80.5x` |

### Rationale

- **Better investigation pivots:** Error and anomaly summaries should contain enough timestamps and context to guide follow-up Loki, Prometheus, or trace queries.
- **Preserved compression:** Info logs and normal high-cardinality metric series should remain compact.
- **Clearer metric semantics:** Counter summaries should report rate and delta rather than raw monotonic growth.
- **Deferred label filtering:** The measured compression gain is already strong, so label filtering should wait for operational evidence.
- **Stable architecture:** The Go fail-open contract and Rust helper boundary from ADR 021 remain unchanged.

## Consequences

### Positive

- **Higher diagnostic value:** Summaries preserve the fields most useful for root-cause investigation.
- **Cleaner agent reasoning:** Agents can focus on errors, anomalies, and compact normal-volume summaries.
- **Better metric interpretation:** Counters, gauges, and anomalies have different summary shapes aligned to their operational meaning.
- **Measured ROI:** Structured summaries preserved strong compression for both logs and metrics without requiring benchmark script changes.

### Negative

- **More complex schema:** Structured entries are more expressive than strings, but require more careful tests and downstream handling.
- **Slightly larger summaries:** Error and anomaly entries may use more bytes than string-only summaries.
- **Deferred filtering decision:** Label filtering may still be needed later for privacy or readability, but it is no longer justified as an immediate compression requirement.

## Verification

- [x] **ADR Review:** ADR 022 captured the structured summary policy before implementation began.
- [x] **Log Tests:** Rust tests cover structured log entries, severity-specific detail, timestamps, and context extraction.
- [x] **Metric Tests:** Rust tests cover vectors, gauges, counters, p99, and reset detection.
- [x] **Fail-Open Check:** `go test ./internal/mcp/tools/telemetry` verified the Go MCP telemetry boundary still accepts summarized output and preserves fail-open behavior.
- [x] **Benchmark Check:** `bench_sidecar_roi.sh --type logs` and `bench_sidecar_roi.sh --type metrics` verified raw bytes, summary bytes, estimated tokens, savings, and density gain.
