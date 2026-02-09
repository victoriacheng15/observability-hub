# RCA 001: Grafana Dashboard Provisioning Failure

- **Status:** ✅ Resolved
- **Date:** 2026-02-09
- **Severity:** 🟡 Medium
- **Author:** Victoria Cheng

## Summary

Grafana was unable to load dashboards after a service restart. The pod initially hung in the `Init` phase due to a missing ConfigMap, and subsequent attempts to provide the configuration resulted in JSON parsing errors within Grafana, rendering the monitoring dashboards inaccessible.

## Timeline

- **2026-02-09 21:40:** Incident detected; Grafana pod stuck in `Init:0/1` phase.
- **2026-02-09 21:42:** Investigation revealed `grafana-dashboards` ConfigMap was missing from the cluster.
- **2026-02-09 21:45:** Manual ConfigMap (`dashboards.yaml`) created with embedded JSON. Service restored but dashboards failed to load.
- **2026-02-09 21:51:** Logs identified `invalid character '\n' in string literal`, indicating malformed JSON in the ConfigMap.
- **2026-02-09 21:58:** Root cause identified: Manual syncing of JSON into YAML is brittle and prone to escaping errors.
- **2026-02-09 22:00:** Permanent fix deployed: Orchestration updated to generate ConfigMap directly from source JSON files.

## Root Cause Analysis

The primary cause was **Configuration Fragility**. Embedding complex JSON (with nested quotes and newlines) inside a YAML block scalar (`|`) for a Kubernetes ConfigMap is error-prone. 

When the JSON was manually copied into the YAML manifest:
1.  Hidden newline characters were injected into the `rawSql` strings.
2.  YAML's interpretation of block scalars and JSON's strict requirement for escaped `\n` characters conflicted.
3.  Grafana's dashboard provider rejected the entire JSON file upon encountering the first unescaped newline within a string.

## Lessons Learned

- **Avoid Manual Syncing:** Never manually embed complex file types (JSON, XML, Scripts) into YAML manifests if they can be mounted as external files.
- **Tooling over Manual Effort:** Use `kubectl create configmap --from-file` to let the Kubernetes API handle the encapsulation and escaping of file contents.
- **Log Visibility:** Grafana's internal provisioning logs provided the "smoking gun" for the parsing error; always check application-level logs when a service is "Running" but malfunctioning.

## Action Items

- [x] **Fix:** Updated `makefiles/k3s.mk` to use `kubectl create configmap --from-file`.
- [x] **Prevention:** Deleted the manual `k3s/grafana/dashboards.yaml` to prevent future maintenance errors.
- [ ] **Process:** Audit other ConfigMaps in the `k3s/` directory for similar embedded-data patterns.

## Verification

- [x] **Manual Check:** Verified dashboards are visible and querying PostgreSQL successfully in the Grafana UI.
- [x] **Automated Tests:** `make k3s-grafana-up` now successfully provisions dashboards from the source directory.
