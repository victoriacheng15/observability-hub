# Network API Specifications

Detailed input schemas for Cilium and Hubble-related MCP tools.

## Tools

### observe_network_flows

- **Description:** Real-time query of flow data via Hubble Relay.
- **Input:**
  - `namespace` (string, optional): Filter by source/destination namespace.
  - `pod` (string, optional): Filter by source/destination pod.
  - `reserved` (string, optional): Filter by reserved entity (e.g. "host", "world", "ingress").
  - `last` (number, optional): Number of recent flows to return (default 20).
- **Returns:** List of flows including Verdict (Forwarded/Dropped), Protocol, Source/Destination labels, and Port.

### query_metrics (via Telemetry)

- **Description:** Use this to query summarized Hubble metrics (e.g. `hubble_drop_total`).
- **Input:**
  - `query` (string): PromQL expression.
- **Returns:** JSON result from Thanos/Prometheus.
