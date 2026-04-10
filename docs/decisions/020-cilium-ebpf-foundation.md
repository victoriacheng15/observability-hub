# ADR 020: Cilium eBPF Foundation

- **Status:** Accepted
- **Date:** 2026-03-16
- **Author:** Victoria Cheng

## Context and Problem Statement

The platform is entering a phase of deep technical introspection. While the default K3s networking (Flannel/iptables) is functional, it operates as a "black box" that obscures the underlying flow of data between services. To evolve as a high-fidelity observability hub, there is a requirement to move beyond basic connectivity and master **eBPF-native networking**.

The primary driver for this decision is the educational opportunity to:

- **Understand Network Internals**: Gain direct visibility into how the kernel handles packets without the abstraction of legacy iptables.
- **Master Cilium & Hubble**: Learn to operate and troubleshoot industry-standard eBPF infrastructure.
- **Eliminate "Kernel Noise"**: Observe how a high-performance datapath behaves during hardware stress tests.
- **Enable L7 Telemetry**: Transition from simple byte-counting to understanding application-level protocols like MQTT at the network layer.

This transition represents an architectural shift from a functional "black box" towards a high-fidelity networking foundation that allows for deeper correlation between network activity and hardware telemetry.

## Decision Outcome

Replace Flannel/iptables with **Cilium** as the CNI for the K3s cluster. This transition establishes an eBPF-native networking foundation to enable deep L7 visibility and high-fidelity telemetry.

### Rationale

- **O(1) Performance**: Eliminates the linear overhead of iptables rules, ensuring network performance remains consistent regardless of the number of services or pods.
- **L7 MQTT Visibility**: Leverages Cilium's kernel-level MQTT parser to attribute network traffic to specific topics without requiring sidecars or application-level instrumentation.
- **Kernel-Level Introspection**: Provides Hubble for deep network flow analysis and Prometheus metric export, essential for long-term efficiency tracking.
- **GreenOps Baseline**: Enables the correlation of bytes/messages processed with power draw (via Kepler) to calculate precise efficiency metrics.

## Consequences

### Positive

- **Network Efficiency**: Significant reduction in "Kernel Noise" and CPU overhead during high-concurrency network operations.
- **Deep Observability**: Direct visibility into L7 protocols (MQTT) at the datapath level.
- **Hardware Correlation**: Direct link between network throughput and hardware power consumption.
- **Enhanced Security**: Ability to implement advanced eBPF-based network policies for domain isolation.

### Negative

- **Kernel Dependency**: Requires a modern kernel with eBPF and BTF support.
- **Operational Complexity**: Involves a non-trivial migration from Flannel to Cilium, requiring cluster-level configuration changes.
- **Resource Overhead**: Adds Cilium and Hubble components to the cluster resource footprint.

## Verification

- [ ] **CNI Status**: Cilium pods are running and managing network interfaces across all nodes.
- [ ] **L7 Visibility**: MQTT flows are verified and visible within the Hubble UI/CLI.
- [ ] **Metrics Integration**: Cilium and Hubble metrics are successfully scraped by Prometheus and visible in Grafana.
- [ ] **Kernel Compliance**: Verification that the host kernel supports necessary eBPF features (BTF, etc.).
