# Incident Reports (RCA)

This directory contains the **Root Cause Analysis (RCA)** and post-mortem reports for service disruptions, bugs, or security incidents within the Observability Hub.

---

## 📂 Incident Log

| RCA | Date | Title | Severity | Status |
| :--- | :--- | :--- | :--- | :--- |
| **002** | 2026-02-12 | [Service Graph Metrics Failure](./002-service-graph-metrics-failure.md) | 🟡 Medium | ✅ Resolved |
| **001** | 2026-02-09 | [Grafana Dashboard Provisioning Failure](./001-grafana-dashboard-provisioning-failure.md) | 🟡 Medium | ✅ Resolved |

---

## 🛠️ Process & Standards

We document incidents to prevent recurrence and build a more resilient system.

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
