# Pod API Specifications

Detailed input schemas for Kubernetes-related MCP tools.

## Tools

### inspect_pods

- **Input:**
  - `namespace` (string): Target namespace (e.g. "default").
- **Returns:** A summary list of pods including their name, status, IP, and node.

### describe_pod

- **Input:**
  - `namespace` (string): Pod's namespace.
  - `name` (string): Name of the pod.
- **Returns:** Full Kubernetes pod specification and status.

### list_pod_events

- **Input:**
  - `namespace` (string): Pod's namespace.
  - `name` (string): Name of the pod.
- **Returns:** List of Kubernetes events (Warnings and Information) associated with the pod.

### get_pod_logs

- **Input:**
  - `namespace` (string): Pod's namespace.
  - `name` (string): Name of the pod.
  - `container` (string, optional): Specific container within the pod.
  - `tail_lines` (number, optional): Number of recent log lines to fetch.
  - `previous` (boolean, optional): If true, fetch logs from the previous container instance (useful for crash analysis).
- **Returns:** Raw log stream.

### delete_pod

- **Input:**
  - `namespace` (string): Pod's namespace.
  - `name` (string): Name of the pod.
  - `grace_seconds` (number, optional): Time period for graceful termination.
- **Returns:** Status confirmation.
