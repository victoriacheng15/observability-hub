---
name: platform
description: Specialized tools for inspecting the overall health of the Observability Hub platform. Use this for executive summaries and checking the health status of core platform components.
---

# Platform Orchestration Skill

This skill provides an executive view of the overall platform health, correlating statuses across all hub components to determine if the system is operational.

## 🛠 Available Tools

| Tool | Purpose | Input Schema |
| :--- | :--- | :--- |
| `hub_inspect_platform` | Get an executive summary of the entire platform health | `{}` |

## 📋 Standard Workflows

### 1. Platform Health Audit

Use `hub_inspect_platform` as the first step when a global outage is suspected. It checks the health of the orchestration engine, databases, and core collectors to provide a high-level status (Healthy/Degraded/Critical).

## 💡 Operational Tips

- **Correlation:** This tool is a high-level aggregator. If it reports a failure, dive deeper into the `host`, `pods`, or `telemetry` skills for specific root causes.

---
*For detailed API documentation, see [references/api-specs.md](references/api-specs.md).*
