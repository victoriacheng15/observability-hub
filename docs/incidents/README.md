# Incident Reports (RCA)

This directory contains the **Root Cause Analysis (RCA)** and post-mortem reports for service disruptions, bugs, or security incidents within the Observability Hub.

---

## 📂 Incident Log

| RCA | Date | Title | Severity | Status |
| :--- | :--- | :--- | :--- | :--- |
| **007** | 2026-04-06 | [Worker Ingestion Blocked from MongoDB Atlas](./007-worker-ingestion-atlas-egress-block.md) | 🟡 Medium | ✅ Resolved |
| **006** | 2026-04-01 | [Loki Gateway DNS Timeout](./006-loki-gateway-dns-timeout.md) | 🟡 Medium | ✅ Resolved |
| **005** | 2026-03-30 | [Kustomize RBAC Resource Collision](./005-kustomize-rbac-resource-collision.md) | 🟡 Medium | ✅ Resolved |
| **004** | 2026-03-17 | [SSH Lockout via Cilium IPAM Collision](./004-ssh-lockout-cilium-ipam-collision.md) | 🔴 Critical | ✅ Resolved |
| **003** | 2026-02-22 | [Thanos Discovery and Retention Failure](./003-thanos-discovery-and-retention-failure.md) | 🟡 Medium | ✅ Resolved |
| **002** | 2026-02-12 | [Service Graph Metrics Failure](./002-service-graph-metrics-failure.md) | 🟡 Medium | ✅ Resolved |
| **001** | 2026-02-09 | [Grafana Dashboard Provisioning Failure](./001-grafana-dashboard-provisioning-failure.md) | 🟡 Medium | ✅ Resolved |

---

## 🛠️ Process & Standards

Incident documentation prevents recurrence and builds system resilience.

### ⚖️ When to write an RCA (The Rule of Three)

Formal RCAs are required only if **one or more** of these conditions are met:

1. **Utility Loss**: Failure to fulfill primary purpose (e.g., dashboards inaccessible, telemetry collection halted).
2. **Data Integrity**: Permanent loss, corruption, or unauthorized exposure of data.
3. **Regression (The "Zombie Bug")**: The failure has occurred previously. Identification of the gap in the previous fix is required.

*Minor configuration drifts or "noisy" logs that do not impact system health should be handled via standard Git commit documentation rather than an RCA.*

### Severity Levels

| Level | Meaning |
| :--- | :--- |
| **🔴 High** | Service down, data loss, or security breach. |
| **🟡 Medium** | Partial degradation, performance issues, or feature malfunction. |
| **🔵 Low** | Minor bugs, cosmetic issues, or non-critical failures. |

### Status

| Status | Meaning |
| :--- | :--- |
| **🚧 Investigating** | Identifying the root cause. |
| **🩹 Mitigated** | Temporary fix applied, service restored. |
| **✅ Resolved** | Root cause identified and permanent fix implemented. |

### 📝 RCA Template

To document a new incident, create a new file named `XXX-descriptive-title.md`.

```markdown
# RCA [XXX]: [Descriptive Title]

- **Status:** Investigating | Mitigated | Resolved
- **Date:** YYYY-MM-DD
- **Severity:** High | Medium | Low
- **Author:** Victoria Cheng

## Summary

A brief overview of what happened, the impact, and the duration.

## Timeline

- **YYYY-MM-DD HH:MM:** Incident detected.
- **YYYY-MM-DD HH:MM:** Investigation started.
- **YYYY-MM-DD HH:MM:** Mitigation applied.
- **YYYY-MM-DD HH:MM:** Root cause identified.
- **YYYY-MM-DD HH:MM:** Permanent fix deployed.

## Root Cause Analysis

Detailed explanation of why the incident happened (The "Why").

## Lessons Learned (Optional)

What went well? What went wrong? What did we get lucky with?

## Action Items

- [ ] **Fix:** Immediate technical resolution.
- [ ] **Prevention:** Changes to prevent recurrence (e.g., monitoring, tests).
- [ ] **Process:** Changes to workflows or documentation.

## Verification

- [ ] **Manual Check:**
- [ ] **Automated Tests:**
```
