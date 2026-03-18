# RCA 003: SSH Lockout via Cilium IPAM Collision

- **Status:** ✅ Resolved
- **Date:** 2026-03-17
- **Severity:** 🔴 Critical
- **Author:** Victoria Cheng

## Summary

Following the deployment of the Cilium CNI with `kubeProxyReplacement` enabled, SSH connectivity to the host (`server2`) was lost. The root cause was identified as a CIDR collision between the Cilium Pod IP pool and the host's physical network (`10.0.0.0/24`).

## Timeline

- **2026-03-17 00:10:** Cilium CNI deployed via OpenTofu with default settings.
- **2026-03-17 00:15:** Incident detected; SSH access to `server2` failed with timeout.
- **2026-03-17 00:25:** Investigation via local CLI agent identified CIDR conflict in Cilium ConfigMap (`10.0.0.0/8`).
- **2026-03-17 00:35:** **Root Cause Identified**: eBPF datapath installed a route for `10.0.0.0/24` on the `cilium_host` interface, black-holing return SSH traffic.
- **2026-03-17 00:45:** Initial fix attempted (CIDR correction); encountered `hostPort` conflicts with Cilium Operator.
- **2026-03-17 01:05:** Secondary issue detected: Pods in `CrashLoopBackOff` due to stale BPF state rejecting UDP 53 traffic.
- **2026-03-17 01:15:** Strategy pivoted to full Cilium revert to restore cluster stability.
- **2026-03-17 01:25:** Permanent fix deployed: Manual BPF filesystem cleanup, `iptables` flush, and K3s restart.

## Root Cause Analysis

The primary cause was an **IP Address Space Collision** triggered by default CNI configuration.

1. **Overlapping CIDRs**: Cilium defaulted to `10.0.0.0/8`, which overlapped with the host's physical network (`10.0.0.0/24`).
2. **eBPF Routing Hijack**: The eBPF datapath overrode the host's main routing table, causing response packets for SSH to be sent to internal cluster interfaces.
3. **State Persistence**: Deleting the Kubernetes resource (Helm release) did not remove the kernel-level eBPF maps, leading to persistent `operation not permitted` errors for pod egress traffic.

## Lessons Learned

- **Kernel State != Kubernetes State**: High-privilege components like Cilium require manual kernel cleanup (BPF/iptables) during a revert.
- **CIDR Awareness is Mandatory**: Always explicitly define Pod CIDRs to ensure they are mathematically separated from host management networks.
- **Single-Node Constraints**: Rolling updates using `hostPort` (like Cilium Operator) will fail on single-node clusters unless replicas are set to 1.

## Action Items

- [x] **Fix**: Reverted Cilium deployment and restored Flannel networking.
- [x] **Documentation**: Created `tofu/CILIUM_RECOVERY_GUIDE.md` for future kernel-level cleanup.
- [ ] **Prevention**: Implement a CIDR validation script in the IaC pipeline.

## Verification

- [x] **SSH Connectivity**: Verified external access to port 22 is functional.
- [x] **Pod Readiness**: Confirmed all observability and hub pods are `Ready 1/1`.
- [x] **DNS Resolution**: Verified `coredns` resolution from within application pods.
