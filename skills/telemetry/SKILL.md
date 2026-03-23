---
name: telemetry
description: Specialized tools for querying the LGTM stack (Thanos, Loki, Tempo) and performing automated incident investigations. Use this when the user asks to analyze metrics, search logs, retrieve traces, or run a full incident correlation.
---

# Telemetry Investigator Skill

This skill provides a high-efficiency interface to the Observability Hub's telemetry tools. It abstracts the underlying PromQL, LogQL, and Trace ID retrieval logic.

## 🛠 Available Tools

| Tool | Purpose | Input Schema |
| :--- | :--- | :--- |
| `query_metrics` | Execute PromQL against Thanos/Prometheus | `{ "query": "string" }` |
| `query_logs` | Execute LogQL against Loki | `{ "query": "string", "limit": number }` |
| `query_traces` | Retrieve distributed traces from Tempo | `{ "trace_id": "string" }` |
| `investigate_incident` | Correlate all signals for a service | `{ "service": "string", "hours": number }` |

## 📋 Standard Workflows

### 1. The "Three Pillars" Correlation

When a service is reported as degraded:

1. Run `query_metrics` to check error rates/latency (`sum(rate(http_requests_total{status=~"5.."}[5m]))`).
2. Run `query_logs` with the same time range to find specific error messages.
3. If a `trace_id` is found in logs, use `query_traces` to identify the bottleneck span.

### 2. Autonomous Investigation

Use `investigate_incident` as a macro-tool for rapid RCA. It automatically performs the correlation above and produces a markdown report.

## 💡 Query Tips

- **Loki:** Use `{job="service-name"}` for targeted log searches.
- **Thanos:** Use `rate(...)` for error percentages rather than absolute counts.
- **Tempo:** Trace IDs are usually 32-character hex strings found in log metadata.

---
*For detailed API documentation, see [references/api-specs.md](references/api-specs.md).*
