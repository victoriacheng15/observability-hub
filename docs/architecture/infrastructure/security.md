# Security Architecture

The Observability Hub employs a multi-layered security model to protect the data pipeline while maintaining external accessibility for webhooks.

## 📡 External Ingress (Tailscale Funnel)

GitHub webhooks are received through **Tailscale Funnel** without exposing the entire server to the public internet.

- **Scoped Exposure**: Only port `8443` (HTTPS) is exposed to the public.
- **Termination**: TLS is terminated at the Tailscale edge; traffic is forwarded to the local Proxy service over the secure Tailscale mesh.
- **Dynamic Gating**: The `tailscale-gate` service ensures the funnel is closed if the Proxy is inactive.

## 🔐 Webhook Authentication

All incoming requests to `/api/webhook/gitops` must be authenticated using **HMAC-SHA256 signature verification**.

1. **Secret Storage**: The `GITHUB_WEBHOOK_SECRET` is stored in the `.env` file (not committed to Git).
2. **Verification**: The Proxy validates the `X-Hub-Signature-256` header against the request body before any processing occurs.
3. **Event Filtering**: Only specifically defined events (`push` or `pull_request` to `main`) are processed; all others are rejected early to minimize resource consumption.

## 🧪 Hybrid Isolation

- **eBPF-Native Isolation**: The platform leverages **Cilium** for kernel-level network isolation. By replacing legacy iptables with an eBPF-native datapath, the system enforces security policies with $O(1)$ efficiency, ensuring that the performance of the observability pipeline is not degraded by the number of active rules.
- **Domain-Based Policies**: Network security is enforced via **CiliumNetworkPolicies**, which allow for granular, L7-aware isolation. This ensures that only authorized services (like the Proxy) can communicate with internal data engines, while unauthorized lateral movement is blocked at the kernel level.
- **Service Boundaries**: The Proxy (running as a native systemd service) acts as the primary "bridge" between the public funnel and the internal cluster data tier.
- **Environment Variables**: Sensitive credentials (passwords, URIs) are managed via **Kubernetes Secrets** or retrieved from OpenBao, ensuring they never appear in plain text in process lists.

## 🧰 Workload Hardening

Cluster workloads are hardened at the image and manifest layers before ArgoCD reconciles them into the runtime.

- **Non-Root Containers**: Project-built images for the chaos controller, sensor fleet, and worker create and run as a non-root UID/GID instead of relying on root defaults.
- **Pod and Container Security Contexts**: Kubernetes manifests define `runAsNonRoot`, explicit user and group IDs, `seccompProfile: RuntimeDefault`, `allowPrivilegeEscalation: false`, and `capabilities.drop: ["ALL"]` where supported.
- **Read-Only Root Filesystems**: Hardened workloads set `readOnlyRootFilesystem: true`. Required write paths are modeled explicitly with PVCs or `emptyDir` mounts, such as n8n state, cache, and temporary directories.
- **Service Account Boundaries**: Workloads that do not need Kubernetes API access disable token automounting. Workloads that do need API access, such as the chaos controller, keep a scoped service account and RBAC role.

## 🧱 Repository Integrity

The GitOps agent (`gitops_sync.sh`) uses **Fast-Forward Only (`--ff-only`)** merges to prevent accidental merge commits on the host and ensure the local environment stays strictly in sync with the remote "Source of Truth."

## 🤖 Automated Security Governance

To maintain a "Secure by Default" posture, the repository employs automated CI/CD guardrails:

- **GitOps Desired State**: **ArgoCD** continuously enforces the desired state defined in Git, automatically reverting any unauthorized manual modifications to Kubernetes resources.
- **Infrastructure Linting**: Every change to the `k3s/` directory is automatically scanned by `kube-linter` to identify misconfigurations and security violations in Kubernetes manifests.
- **Misconfiguration Scanning**: **Trivy** scans Dockerfiles and Kubernetes manifests for HIGH and CRITICAL configuration risks, including root containers, missing security contexts, and mutable root filesystems.
- **Vulnerability Scanning**: The Go codebase and its dependencies are continuously audited using `govulncheck` (triggered on pushes and weekly schedules) to proactively identify and remediate known vulnerabilities.
- **Policy Enforcement**: Markdown and HCL configurations are linted to ensure consistent adherence to operational and security standards across all documentation and policies.
