# Cilium & Hubble: Network Layer Intelligence

This guide explains how the Observability Hub uses **Cilium (eBPF)** and **Hubble** to visualize and secure the network workflow across L3, L4, and L7 layers.

## 1. The Three Layers of Visibility

Cilium moves network logic into the Linux kernel using eBPF, allowing us to see traffic in three distinct stages of meaning:

| Layer | Focus | Hubble UI Representation | Example Discovery |
| :--- | :--- | :--- | :--- |
| **L3 (Network)** | **"The Who"** | Identity Labels (e.g., `otel-collector`) | "Pod A is talking to Pod B." |
| **L4 (Transport)** | **"The How"** | Ports & Verdicts (e.g., `3100`, `forwarded`) | "The connection is healthy and authorized." |
| **L7 (Application)** | **"The What"** | Protocols & Paths (e.g., `POST /loki/push`) | "The application is sending logs to Loki." |

---

## 2. Operationalizing the Workflow

Hubble transforms raw binary traffic into a **Logical Service Map**. Use this map to identify three critical system states:

### 🟢 Green Lines (Forwarded)
- **Meaning**: The kernel has verified the identity of both pods and an explicit **CiliumNetworkPolicy** (CNP) or **ClusterwidePolicy** (CCNP) allows the flow.
- **Action**: None required. The "Pipe" is open.

### 🔴 Red Lines (Dropped)
- **Meaning**: A **Security Violation**. Cilium is in "Default-Deny" mode because a policy exists in that namespace, but no rule allows this specific flow.
- **Action**: Check `hubble_drop_total` metrics or pod logs. Usually requires updating the `endpointSelector` or `toPorts` in the policy.

### 🟡 Yellow/Dash Lines (L7 Errors)
- **Meaning**: The connection (L4) is allowed, but the application (L7) is failing (e.g., `HTTP 500` or `gRPC Timeout`).
- **Action**: Investigate the destination pod's logs or check **Tempo Traces** for the specific `traceID`.

---

## 3. Deep Visibility (L7 Proxy)

By default, Cilium is "Content-Blind" to save CPU. In this repo, we enable **Deep Visibility** using the following architecture:

1.  **Envoy Proxy**: Enabled in `tofu/02-networking.tf` via `l7Proxy = true`. This starts a sidecar-less proxy on each node.
2.  **Visibility Policies**: Policies like `observability-l7-global-visibility` tell the proxy which ports to "peel open."
    - **Port 3100**: Loki (HTTP)
    - **Port 4317/4318**: OTel/Tempo (gRPC/HTTP)
    - **Port 9090**: Prometheus (HTTP)

---

## 4. Policy Hierarchy: CNP vs. CCNP

| Type | Scope | Usage in this Hub |
| :--- | :--- | :--- |
| **CNP** | Namespace-only | Used for local, internal service hardening. |
| **CCNP** | **Cluster-wide** | Used for the Observability Stack to allow telemetry from **all namespaces** (Databases, Sensors, etc.). |

### Pro-Tip: The "L7 Cheat Code"
If you don't see L7 info in Hubble but need to know "What" is moving, use the **MCP Agents** to query **Tempo**:
```bash
# Ask the agent:
"Show me the latest traces for the proxy service"
```
Traces often show the same L7 info (URLs/Paths) as Hubble but with even more internal application context.
