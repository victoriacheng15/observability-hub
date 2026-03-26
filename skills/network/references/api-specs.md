# Network API Specifications

Detailed input schemas for Cilium and Hubble-related MCP tools.

## Tools

### observe_network_flows

- **Description:** Real-time query of flow data via Hubble Relay.
- **Input:**
  - `namespace` (string, optional): Filter by source or destination namespace.
  - `pod` (string, optional): Filter by source or destination pod.
  - `from_pod` (string, optional): Filter by source pod name prefix. Use `[namespace/]<pod-name>` format.
  - `to_pod` (string, optional): Filter by destination pod name prefix. Use `[namespace/]<pod-name>` format.
  - `protocol` (string, optional): Filter by L4/L7 protocol (e.g., "tcp", "udp", "http").
  - `port` (number, optional): Filter by source or destination port.
  - `to_port` (number, optional): Filter by destination port.
  - `verdict` (string, optional): Filter by verdict (`FORWARDED`, `DROPPED`, `AUDIT`, `REDIRECTED`).
  - `http_status` (string, optional): Filter by HTTP status code prefix (e.g., "404", "5+").
  - `http_method` (string, optional): Filter by HTTP method (e.g., "GET", "POST").
  - `http_path` (string, optional): Filter by HTTP path regular expression.
  - `reserved` (string, optional): Filter by reserved entity (e.g., "host", "world", "ingress").
  - `last` (number, optional): Number of recent flows to return (default 20, max 100).
- **Returns:** List of JSON-formatted flows including Verdict, Protocol, Source/Destination identity, and L7 details if applicable.

### query_metrics (via Telemetry)

- **Description:** Use this to query summarized Hubble metrics (e.g. `hubble_drop_total`).
- **Input:**
  - `query` (string): PromQL expression.
- **Returns:** JSON result from Thanos/Prometheus.
