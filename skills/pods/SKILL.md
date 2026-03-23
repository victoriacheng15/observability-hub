---
name: pods
description: Specialized tools for managing Kubernetes pods within the Observability Hub. Use this for inspecting pod health, retrieving logs, and troubleshooting container-level issues.
---

# Pod Management Skill

This skill provides direct access to the Kubernetes cluster's pod lifecycle and operational data. It abstracts the `kubectl` complexity into high-signal tools for agentic analysis.

## 🛠 Available Tools

| Tool | Purpose | Input Schema |
| :--- | :--- | :--- |
| `inspect_pods` | List all pods in a namespace with status summary | `{ "namespace": "string" }` |
| `describe_pod` | Get detailed status and configuration for a pod | `{ "namespace": "string", "name": "string" }` |
| `list_pod_events` | List all lifecycle events associated with a pod | `{ "namespace": "string", "name": "string" }` |
| `get_pod_logs` | Retrieve logs from a specific pod/container | `{ "namespace": "string", "name": "string", "container": "string", "tail_lines": number, "previous": boolean }` |
| `delete_pod` | Delete a specific pod (useful for restarts) | `{ "namespace": "string", "name": "string", "grace_seconds": number }` |

## 📋 Standard Workflows

### 1. Pod Health Check

When a service is reported as "Down" or "Degraded":

1. Run `inspect_pods` in the relevant namespace to check for `Pending`, `Failed`, or `CrashLoopBackOff` statuses.
2. Use `describe_pod` to identify resource constraints (CPU/Memory limits) or configuration errors.
3. Check `list_pod_events` for recent `BackOff` or `FailedScheduling` events.

### 2. Log Analysis

If a pod is running but behaving incorrectly:

1. Run `get_pod_logs` with a reasonable `tail_lines` (e.g., 100).
2. If the pod has restarted, use `previous: true` to see the logs from the crashed container.

## 💡 Operational Tips

- **Namespace:** Most hub services live in the `default` or `observability` namespaces.
- **Graceful Deletion:** Use `delete_pod` only when a restart is necessary to clear a stuck state.

---
*For detailed API documentation, see [references/api-specs.md](references/api-specs.md).*
