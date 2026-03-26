---
name: network
description: Specialized tools for network-level observability using Cilium and Hubble. Use this to investigate cluster-wide network flows, packet drops, and host-level networking without being restricted to a single namespace.
---

# Network Observability Skill (Cilium/Hubble)

This skill leverages eBPF-powered introspection from Cilium and Hubble to provide deep visibility into the network datapath. It allows for real-time flow analysis and troubleshooting of connectivity issues across L3, L4, and L7.

## 🛠 Available Tools

| Tool | Purpose | Input Schema |
| :--- | :--- | :--- |
| `observe_network_flows` | Query real-time network flows from Hubble Relay | `{ "namespace": "string", "pod": "string", "from_pod": "string", "to_pod": "string", "protocol": "string", "port": number, "to_port": number, "verdict": "string", "http_status": "string", "http_method": "string", "http_path": "string", "reserved": "string", "last": number }` |
| `query_metrics` | (via Telemetry) Execute PromQL for Hubble/Cilium metrics | `{ "query": "string" }` |

## 📋 Standard Workflows

### 1. Cluster-wide Flow Investigation

Unlike the Hubble UI, the `observe_network_flows` tool allows for wider queries:

1. **Host Networking**: Check flows specifically interacting with the host stack by setting `reserved: "host"`.
2. **Directional Traffic**: Use `from_pod` and `to_pod` (format: `[namespace/]<pod-name>`) to audit communication between specific workloads.
3. **Cross-namespace**: Query flows without a namespace filter to see inter-service communication across the entire hub.

### 2. Identifying Network Drops

If a service is failing to connect:

1. Query `hubble_drop_total` via `query_metrics` to see if the kernel is dropping packets.
2. Use `observe_network_flows` with `verdict: "DROPPED"` and `last: 20` to see exactly what traffic is being blocked and why.

### 3. DNS Troubleshooting

1. Check `hubble_dns_queries_total` for high failure rates.
2. Use `observe_network_flows` with `protocol: "udp"` and `port: 53` to verify if DNS traffic is reaching the `kube-dns` pods.

### 4. L7 (HTTP) Auditing

1. Filter for specific failure codes using `http_status: "5+"` to find server-side errors.
2. Monitor specific API routes by setting `http_path: "/api/v1/.*"`.

## 💡 Operational Tips

- **Hubble Relay**: These tools connect to Hubble Relay, meaning they see flows across ALL nodes and namespaces.
- **Prefix Matching**: Pod filters (`pod`, `from_pod`, `to_pod`) use prefix matching. `databases/postgres` will match all postgres instances in the `databases` namespace.
- **L7 Visibility**: Remember that L7 (HTTP/gRPC) visibility requires a `CiliumNetworkPolicy` to be active on the target port.
- **Verdicts**: Common verdicts include `FORWARDED`, `DROPPED`, `AUDIT`, and `REDIRECTED`.

---
*For detailed API documentation, see [references/api-specs.md](references/api-specs.md).*
