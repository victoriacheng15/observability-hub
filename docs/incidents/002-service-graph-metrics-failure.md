# RCA 002: Service Graph Metrics Failure

- **Status:** ✅ Resolved
- **Date:** 2026-02-12
- **Severity:** 🟡 Medium
- **Author:** Victoria Cheng

## Summary

Following the implementation of Trace Diversification, the "Service Graph" and "Breakdown" tabs in Grafana remained empty. While traces were successfully reaching Tempo, the structural metrics (RED metrics) required to render the graph and the breakdown table were not being ingested into Prometheus.

## Timeline

- **2026-02-12 16:45:** Trace diversification implemented in Go Proxy and Traffic Generator.
- **2026-02-12 16:50:** Incident detected; Service Graph in Grafana displayed "No service graph data found."
- **2026-02-12 17:05:** Investigation into OTel Collector identified that the `servicegraph` connector was inactive and would require complex instrumentation (Client + Server spans) to function with `curl`-based traffic.
- **2026-02-12 17:25:** Strategy pivoted to use Tempo's internal `metrics-generator`, which supports "virtual nodes" for uninstrumented clients.
- **2026-02-12 17:42:** **Root Cause Identified**: Tempo logs revealed `404 Not Found` when attempting to Remote Write metrics to Prometheus.
- **2026-02-12 17:48:** Permanent fix deployed: Prometheus updated with `--web.enable-remote-write-receiver` and metric allowlists; Tempo processors explicitly enabled.

## Root Cause Analysis

The primary cause was a **Disconnected Telemetry Pipeline** due to a disabled receiver in the downstream storage engine (Prometheus).

1. **Disabled Sink**: The Prometheus instance was running with default settings, which do not allow external services to "push" metrics via Remote Write.
2. **Inactive Processors**: Tempo's `metrics_generator` was defined in the configuration but was not explicitly activated via the `overrides` section, meaning it was not actually analyzing traces for structural data.
3. **Strict Semantic Requirements**: The OTel Collector's `servicegraph` implementation was too strict for our current testing methodology (direct `curl` calls), whereas Tempo's implementation was better suited for this "single-node" hybrid environment.

## Lessons Learned

- **End-to-End Connectivity is Key**: In a distributed observability system, verifying the "sink" (the database) is just as important as verifying the "source" (the code).
- **Log-Driven Debugging**: The "smoking gun" was found by inspecting the Tempo pod logs, which explicitly stated that Prometheus was refusing the connection.
- **Virtual Nodes for Legacy Traffic**: Using Tempo's metrics-generator is more forgiving than the OTel Collector for traffic originating from uninstrumented tools like `curl`, `wget`, or external scripts.

## Action Items

- [x] **Fix**: Enabled `--web.enable-remote-write-receiver` in `k3s/prometheus/values.yaml`.
- [x] **Fix**: Activated `service-graphs`, `span-metrics`, and `local-blocks` in `k3s/tempo/values.yaml`.
- [x] **Fix**: Enabled `ingester` WAL to support the `local-blocks` processor requirements.
- [x] **Optimization**: Updated Prometheus `traces_.*` allowlist to ensure all structural metrics are persisted.

## Verification

- [x] **Metric Query**: Verified `traces_service_graph_request_total` is present in Prometheus via Port-Forwarding.
- [x] **UI Check**: Confirmed the `proxy` node and the **Breakdown** tab are fully functional in the Grafana Tempo UI.
