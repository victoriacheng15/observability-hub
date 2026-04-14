# ADR 022: Structured Summaries for Obs Processor

- **Status:** Proposed
- **Date:** 2026-04-14
- **Author:** Victoria Cheng

## Context and Problem Statement

ADR 021 introduced `obs-processor`, a Rust helper binary that compresses raw Loki and Prometheus responses before the MCP telemetry tools return them to agents. That decision remains valid: Go owns query validation, provider calls, timeout control, and fail-open behavior, while Rust owns the response reduction step.

Live use showed that the first summary format is too lossy for investigation. It reduces payload size aggressively, but important diagnostic pivots disappear:

- **Repeated log errors lose cause:** An error such as `webhook_sync_failed (x2)` does not preserve structured fields like `repo`, `error`, `output`, or the failure window.
- **Timestamps disappear:** Grouped log entries do not expose first and last timestamps, making it harder to pivot into raw logs and traces around the event.
- **Warnings and info logs are not differentiated enough:** Normal high-volume info logs need minimal counts, while warnings and errors need more diagnostic context.
- **Metric summaries stay noisy for high-cardinality results:** Large Prometheus responses can still emit one summarized entry per series, including long label sets.
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

The target metric summary groups normal high-cardinality series and makes anomalies explicit:

```json
{
  "result_type": "matrix",
  "total_series": 128,
  "total_samples": 7680,
  "summarized_count": 2,
  "entries": [
    {
      "metric": "node_cpu_seconds_total",
      "kind": "counter",
      "status": "normal",
      "labels": {
        "mode": "idle"
      },
      "series_count": 16,
      "sample_count": 960,
      "delta": 54516.25,
      "average_rate_per_second": 15.14,
      "resets_detected": 0,
      "first_timestamp": 1776126896.4,
      "last_timestamp": 1776130436.4
    },
    {
      "metric": "up",
      "kind": "gauge",
      "status": "anomaly",
      "labels": {
        "job": "kubernetes-service-endpoints",
        "instance": "service-endpoint:4244"
      },
      "sample_count": 1,
      "current": 0,
      "expected": 1,
      "first_timestamp": 1776130436.4,
      "last_timestamp": 1776130436.4,
      "context": {
        "reason": "scrape target down"
      }
    }
  ],
  "omitted_series": 126
}
```

### Log Summary Policy

Log summaries remain repetition-aware, but the amount of detail depends on severity:

- **Errors:** Include count, first timestamp, last timestamp, and bounded diagnostic context.
- **Warnings:** Include count, timestamps, and limited context when useful.
- **Info:** Keep minimal by default, usually level, message, and count.

Diagnostic context should be extracted from structured Loki stream metadata through a small allowlist. Useful fields include `service_name`, `repo`, `error`, `status`, `path`, `ref`, `action`, and a bounded `output_preview`.

### Metric Summary Policy

Metric summaries should distinguish normal volume from operational anomalies:

- **Anomalies:** Include status, retained labels, timestamps, current or expected values, and short context.
- **Counters:** Use delta, average rate, reset count, sample count, and timestamps.
- **Gauges:** Use min, max, average, p95, first value, last value, trend delta, sample count, and timestamps.
- **Normal high-cardinality series:** Group by the smallest useful retained label set and report omitted counts instead of listing every series.

Metric labels should be filtered through an allowlist so summary text is not dominated by noisy labels. The processor may retain extra labels for small result sets when they are the only useful differentiator.

### Benchmark Policy

Benchmarking remains required, but benchmark script changes are deferred until after the log and metric refactors land. The benchmark must continue to prove that summarization preserves the practical ROI of the sidecar:

- Raw bytes
- Summary bytes
- Estimated tokens
- Estimated savings
- Context-density gain

The benchmark should validate the final structured schema after implementation rather than being updated before the schema exists.

### Rationale

- **Better investigation pivots:** Error and anomaly summaries should contain enough timestamps and context to guide follow-up Loki, Prometheus, or trace queries.
- **Preserved compression:** Info logs and normal high-cardinality metric series should remain compact.
- **Clearer metric semantics:** Counter summaries should report rate and delta rather than raw monotonic growth.
- **Stable architecture:** The Go fail-open contract and Rust helper boundary from ADR 021 remain unchanged.

## Consequences

### Positive

- **Higher diagnostic value:** Summaries preserve the fields most useful for root-cause investigation.
- **Cleaner agent reasoning:** Agents can focus on errors, anomalies, and compact normal-volume summaries.
- **Better metric interpretation:** Counters, gauges, and anomalies have different summary shapes aligned to their operational meaning.
- **Controlled cardinality:** Label filtering and grouping reduce large metric responses without hiding anomaly context.

### Negative

- **More complex schema:** Structured entries are more expressive than strings, but require more careful tests and downstream handling.
- **Slightly larger summaries:** Error and anomaly entries may use more bytes than string-only summaries.
- **Implementation sequencing:** Logs, metrics, grouping, and benchmark validation should land in small PRs to keep review manageable.

## Verification

- [ ] **ADR Review:** Confirm the structured summary policy is accepted before implementation begins.
- [ ] **Log Tests:** Rust tests cover structured log entries, severity-specific detail, timestamps, and context extraction.
- [ ] **Metric Tests:** Rust tests cover vectors, gauges, counters, reset detection, label filtering, grouping, and anomaly summaries.
- [ ] **Fail-Open Check:** Go MCP handlers still return raw telemetry if the processor fails.
- [ ] **Benchmark Check:** After log and metric refactors land, verify `bench_sidecar_roi.sh` still reports raw bytes, summary bytes, estimated tokens, savings, and density gain.
