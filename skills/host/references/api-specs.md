# Host API Specifications

Detailed input schemas for host-level MCP tools.

## Tools

### hub_inspect_host

- **Input:**
  - `(none)` (Empty object `{}`).
- **Returns:** Real-time host metrics for Load Average, Memory usage, and Disk partition status.

### hub_list_host_services

- **Input:**
  - `(none)` (Empty object `{}`).
- **Returns:** A list of systemd units and their current status (Active, Load, Sub-state).

### hub_query_service_logs

- **Input:**
  - `service` (string): The systemd unit name (e.g. "proxy", "ingestion").
  - `since` (string): Relative lookback duration (e.g. "5m", "1h").
- **Returns:** Raw journal output from systemd.
