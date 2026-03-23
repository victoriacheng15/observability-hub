---
name: network
description: Specialized tools for network-level observability using Cilium and Hubble. Use this to investigate cluster-wide network flows, packet drops, and host-level networking without being restricted to a single namespace.
---

# Network Observability Skill (Cilium/Hubble)

This skill leverages eBPF-powered introspection from Cilium and Hubble to provide deep visibility into the network datapath. It allows for real-time flow analysis and troubleshooting of connectivity issues across L3, L4, and L7.

## 🛠 Available Tools

| Tool | Purpose | Input Schema |
| :--- | :--- | :--- |
| `observe_network_flows` | Query real-time network flows from Hubble Relay | `{ "namespace": "string", "pod": "string", "last": number }` |
| `query_metrics` | (via Telemetry) Execute PromQL for Hubble/Cilium metrics | `{ "query": "string" }` |

## 📋 Standard Workflows

### 1. Cluster-wide Flow Investigation

Unlike the Hubble UI, the `observe_network_flows` tool allows for wider queries:

1. **Host Networking**: Check flows between a pod and the host by looking for the `host` entity in Hubble.
2. **Cross-namespace**: Query flows without a namespace filter to see inter-service communication across the entire hub.

### 2. Identifying Network Drops

If a service is failing to connect:

1. Query `hubble_drop_total` via `query_metrics` to see if the kernel is dropping packets.
2. Use `observe_network_flows` with `last: 20` to see the most recent verdicts (Forwarded vs. Dropped) for a specific pod.

### 3. DNS Troubleshooting

1. Check `hubble_dns_queries_total` for high failure rates.
2. Use `observe_network_flows` to verify if DNS traffic is reaching the `kube-dns` pods.

## 💡 Operational Tips

- **Hubble Relay**: These tools connect to Hubble Relay, meaning they see flows across ALL nodes and namespaces.
- **Filtering**: You can filter by `namespace` or `pod` to reduce noise, but leaving them empty provides a cluster-wide view.
- **L7 Visibility**: Remember that L7 (HTTP/gRPC) visibility requires a `CiliumNetworkPolicy` to be active on the target port.

---
*For detailed API documentation, see [references/api-specs.md](references/api-specs.md).*
