---
name: host
description: Specialized tools for managing the physical host server and systemd services. Use this to inspect CPU/Memory pressure, monitor core hub units, and query journal logs.
---

# Host & System Management Skill

This skill provides direct visibility into the physical server's resource health and the status of systemd-managed hub components.

## 🛠 Available Tools

| Tool | Purpose | Input Schema |
| :--- | :--- | :--- |
| `hub_inspect_host` | Inspect physical resource pressure (Load, Memory, Disk) | `{}` |
| `hub_list_host_services` | List and check status of core systemd units | `{}` |
| `hub_query_service_logs` | Query systemd journal logs for a service | `{ "service": "string", "since": "string" }` |

## 📋 Standard Workflows

### 1. Resource Pressure Investigation

If services are slow but metrics look okay, run `hub_inspect_host` to check for physical bottlenecks:

1. **CPU Load:** Check load averages (1m, 5m, 15m) relative to core count.
2. **Memory:** Check available vs. used memory.
3. **Disk:** Ensure partitions (especially for databases and logs) are below 90% capacity.

### 2. Service Management

For core hub components (Ingestion, Proxy, OpenBao) that run as systemd units:

1. Run `hub_list_host_services` to ensure all units are in the `active (running)` substate.
2. Use `hub_query_service_logs` with `since: "5m"` to view recent logs if a service is restarting.

## 💡 Operational Tips

- **Core Units:** Common units include `ingestion.service`, `proxy.service`, and `openbao.service`.
- **Time Window:** Standard formats for `since` include `10s`, `5m`, or `1h`.

---
*For detailed API documentation, see [references/api-specs.md](references/api-specs.md).*
