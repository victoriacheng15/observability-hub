# RCA 003: Thanos Discovery and Retention Failure

- **Status:** ✅ Resolved
- **Date:** 2026-02-22
- **Severity:** 🟡 Medium
- **Author:** Victoria Cheng

## Summary

Thanos long-term metrics access was degraded by a combination of sidecar discovery, UID/security-context, object storage, and retention configuration issues. Prometheus metrics existed locally, but Thanos Query could not reliably discover the Prometheus sidecar path and the retention pipeline was incomplete.

The repair happened across two fixes: `9663b4c` restored the Thanos sidecar/compactor/storage path and retention settings, while `edfeac0` later corrected Thanos Query discovery by adding an explicit headless service and endpoint flag.

## Timeline

- **2026-02-23 04:11 UTC:** `9663b4c` merged to resolve Thanos UID mismatch and enforce observability retention policies.
- **2026-02-23 04:11 UTC:** Prometheus sidecar configuration changed to a manual Thanos sidecar container with explicit UID/GID settings.
- **2026-02-23 04:11 UTC:** Thanos compactor and retention settings added for raw, 5m, and 1h resolutions.
- **2026-03-11 02:39 UTC:** `edfeac0` merged to restore Thanos discovery and add a traffic generator.
- **2026-03-11 02:39 UTC:** Thanos discovery moved away from chart DNS discovery and added an explicit headless `prometheus-thanos-grpc` service plus query endpoint flag.

## Root Cause Analysis

The primary cause was a **chart abstraction mismatch with the platform's single-node storage and security model**.

1. **Sidecar assumptions did not match runtime permissions**: The default chart sidecar path did not line up cleanly with the Prometheus volume and security context in this cluster.
2. **Discovery was too implicit**: Thanos Query relied on chart DNS discovery behavior that did not reliably locate the Prometheus sidecar endpoint.
3. **Retention pipeline was incomplete**: Without a working compactor and explicit retention settings, long-term metrics behavior was underdefined even if short-term Prometheus scraping continued.
4. **Object storage details mattered**: The MinIO-backed object store required path-style behavior and compatible write permissions for supporting jobs.

## Lessons Learned

- **Long-term metrics need end-to-end validation**: Prometheus being healthy does not prove Thanos sidecar, store gateway, query, compactor, and object storage are healthy.
- **Chart defaults are not architecture guarantees**: Security contexts, PVC ownership, and discovery modes need to be validated against the actual cluster topology.
- **Prefer explicit discovery for single-node critical paths**: A headless service and explicit endpoint flag are easier to reason about than implicit chart DNS discovery when debugging.

## Action Items

- [x] **Fix:** Added a manual Thanos sidecar container with explicit UID/GID and object storage config.
- [x] **Fix:** Enabled Thanos Query and compactor with retention policies.
- [x] **Fix:** Added explicit Prometheus Thanos gRPC service discovery in OpenTofu.
- [x] **Prevention:** Add a smoke check that queries Thanos for recent Prometheus data and verifies sidecar endpoint visibility.
- [x] **Observability:** Add dashboard or alert coverage for Thanos Query endpoint health and compactor lag.

## Verification

- [x] **Config Diff:** Verified `9663b4c` updated Prometheus, Thanos, Loki, Tempo, and MinIO manifests/values for sidecar, retention, and object storage compatibility.
- [x] **Config Diff:** Verified `edfeac0` added the headless `prometheus-thanos-grpc` service and explicit Thanos query endpoint flag.
- [x] **Manual Check:** Confirm Thanos Query returns current Prometheus series through the sidecar endpoint.
- [x] **Automated Tests:** Add a Thanos query smoke test to the observability deployment workflow.
