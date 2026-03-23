# LGTM API Specifications

Detailed input schemas and error codes for the telemetry stack tools.

## Tools

### query_metrics (Thanos/Prometheus)

- **Input:**
  - `query` (string): PromQL expression.
- **Common Queries:**
  - `up{job="<service>"}`: Service availability.
  - `sum(rate(http_requests_total[5m]))`: Throughput.
- **Returns:** JSON result from the Thanos/Prometheus query API.

### query_logs (Loki)

- **Input:**
  - `query` (string): LogQL expression.
  - `limit` (number, default: 100): Maximum results to return.
- **Common Queries:**
  - `{job="<service>"} |= "error"`: Filter errors for a service.
- **Returns:** Formatted list of log streams and entries.

### query_traces (Tempo)

- **Input:**
  - `trace_id` (string): 32-character hex ID.
- **Returns:** Complete trace JSON with span hierarchy.

### investigate_incident (Macro-Tool)

- **Input:**
  - `service` (string): Name of the service to investigate.
  - `hours` (number, default: 1): Time window for investigation.
- **Logic:** This tool performs multi-signal correlation across metrics, logs, and traces. It produces a Markdown RCA (Root Cause Analysis) report.
