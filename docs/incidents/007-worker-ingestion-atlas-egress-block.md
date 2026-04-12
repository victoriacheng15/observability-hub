# RCA 007: Worker Ingestion Blocked from MongoDB Atlas

- **Status:** ✅ Resolved
- **Date:** 2026-04-06
- **Severity:** 🟡 Medium
- **Author:** Victoria Cheng

## Summary

After ingestion was migrated from the legacy host-managed binary to the cluster-native `worker` CronJob, the Cilium egress policy for the `hub` namespace was not updated for MongoDB Atlas. The worker pod had DNS and internal database access, but Atlas replica set connections use TCP `27017`, which was not included in the existing external egress allowlist.

The affected window is inferred from GitOps history and the CronJob schedule: the `MONGO_URI` path was restored to the worker ingestion pod on **2026-04-06 00:23 UTC**, the first scheduled run after that was **2026-04-06 02:00 UTC**, and the permanent Cilium policy fix merged on **2026-04-07 02:27 UTC**. Detailed Loki log payloads for the first failure have aged out of retention; current retained logs still summarize `worker.ingestion` task failures and later successful runs under the `service_name="worker.ingestion"` label.

## Timeline

- **2026-04-02 17:28 UTC:** `worker-ingestion` CronJob introduced in `k3s/base/worker/ingestion-cronjob.yaml` as part of the unified worker migration.
- **2026-04-02 19:51 UTC:** Legacy analytics and ingestion components removed, making the cluster-native worker the active ingestion path.
- **2026-04-02 21:40 UTC:** Hub namespace egress to OpenTelemetry ports `4317` and `4318` added, indicating the first migration-related pod egress gap for short-lived worker telemetry.
- **2026-04-04 23:23 UTC:** Brain ingestion migrated from the host-dependent `gh` CLI to the GitHub REST API, and `GITHUB_TOKEN` was added to the worker CronJob environment.
- **2026-04-06 00:23 UTC:** `MONGO_URI` restored to the ingestion CronJob environment, re-enabling the code path that connects to MongoDB Atlas from inside the pod.
- **2026-04-06 02:00 UTC:** First likely affected scheduled ingestion run after `MONGO_URI` restoration. The worker pod attempted Atlas access from the `hub` namespace while the Cilium policy still lacked TCP `27017` egress.
- **2026-04-07 02:00 UTC:** Second likely affected scheduled run occurred before the policy fix was merged.
- **2026-04-07 02:19 UTC:** Fix committed to add Atlas egress for worker ingestion.
- **2026-04-07 02:27 UTC:** Permanent fix merged: `*.mongodb.net` FQDN egress and TCP `27017` world egress added to the Cilium policy.
- **2026-04-08 02:00 UTC:** Next scheduled run after the policy fix was expected to use the restored Atlas path successfully.

## Root Cause Analysis

The primary cause was an **orchestration boundary change without a matching network policy update**.

1. **Execution context changed**: Ingestion moved from a host-tier binary to a K3s CronJob running as a pod in the `hub` namespace.
2. **Policy assumptions stayed host-centric**: The existing Cilium policy allowed common outbound web/API ports such as `80`, `443`, `465`, and `587`, but did not include MongoDB Atlas replica set traffic on TCP `27017`.
3. **Atlas access was restored after the migration**: Once `MONGO_URI` was added back to the worker pod environment, the ingestion task could reach the MongoDB code path, but Cilium still denied the required external database connection.
4. **Detection depended on runtime logs**: The manifest change was valid Kubernetes YAML, so this failure did not surface at build or apply time. It only appeared when the scheduled worker attempted the Atlas connection.

## Related Fixes Reviewed

- **`3fee07f` - OTLP egress:** Relevant as an earlier symptom of the same migration class. Short-lived worker pods needed explicit Cilium egress for telemetry ports after moving into K3s, but this did not cause the Atlas failure.
- **`b541d13` - GitHub REST API:** Relevant context. This removed the host dependency on `gh` CLI and added `GITHUB_TOKEN` to the CronJob, making brain ingestion pod-native.
- **`c0c1549` - host service cleanup:** Adjacent but not part of the Atlas outage. It removed `ingestion.service` from host service inspection after the systemd-to-worker transition.
- **`2367de0` - Mongo connection path:** Directly relevant. Restoring `MONGO_URI` activated the Atlas-backed reading ingestion path from inside the worker pod, exposing the missing TCP `27017` egress policy.
- **`46fe367` - Atlas egress:** Direct fix. This added `*.mongodb.net` and TCP `27017` egress to `k3s/cilium-policies/observability-stack-policy.yaml`.

## Lessons Learned

- **Network policy must follow workload ownership changes**: Moving a workload from host-tier systemd into K3s changes the enforcement point. External dependencies need to be revalidated as pod egress, not assumed from the old host path.
- **Secret restoration can expose latent policy gaps**: Adding `MONGO_URI` back to the pod was necessary, but it also activated an external dependency that the policy did not yet allow.
- **Retention limits matter during RCA**: The detailed failure payloads were no longer available by the time this RCA was written. Commit history and retained service-level summaries were enough to bound the incident, but not enough to recover the exact application error line.

## Action Items

- [x] **Fix:** Added Cilium egress for `*.mongodb.net` and TCP `27017` for worker ingestion.
- [x] **Documentation:** Captured the inferred affected window and migration failure mode in this RCA.
- [x] **Prevention:** Add a deployment checklist item for every host-to-pod migration: enumerate external dependencies and update Cilium policy in the same change set.
- [x] **Prevention:** Add a lightweight post-deploy smoke check for `worker.ingestion` that validates DNS and TCP connectivity to Atlas before relying on the daily schedule.
- [x] **Observability:** Keep enough raw worker error logs to cover at least one full weekly ingestion/debug cycle, or export failed CronJob summaries into a longer-retention incident signal.

## Verification

- [x] **Policy Diff:** Verified `46fe367` added the MongoDB Atlas egress block to `k3s/cilium-policies/observability-stack-policy.yaml`; the current manifest allows `*.mongodb.net` and TCP `27017` for worker ingestion.
- [x] **Runtime Signal:** Queried Loki with `service_name="worker.ingestion"`; retained logs include task failures and later successful ingestion completions, but no longer include the original detailed failure payload.
- [x] **Manual Check:** Trigger a one-off `worker-ingestion` Job and confirm Atlas-backed tasks complete successfully.
- [x] **Automated Tests:** Add policy validation or connectivity smoke coverage for external worker dependencies.
