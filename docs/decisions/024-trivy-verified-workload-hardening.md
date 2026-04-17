# ADR 024: Trivy-Verified Workload Hardening

- **Status:** Accepted
- **Date:** 2026-04-17
- **Author:** Victoria Cheng

## Context and Problem Statement

The platform already used GitOps, OpenBao, Cilium policies, kube-linter, and dependency scanning as security controls. The remaining gap was workload-level hardening across project Dockerfiles and Kubernetes manifests.

Trivy reported HIGH configuration findings for several workloads:

- Dockerfiles did not explicitly switch to non-root users.
- Kubernetes workloads relied on default pod and container security contexts.
- Containers did not consistently set read-only root filesystems.

These findings matter because the platform runs long-lived automation and observability workloads in a shared homelab Kubernetes environment. A compromised container should have the smallest practical filesystem, privilege, and identity surface. The control also needs to remain operable under GitOps: security settings should live in source control and be verified before ArgoCD reconciles them.

## Decision Outcome

Adopt Trivy-verified workload hardening as the baseline for project-managed Dockerfiles and Kubernetes workloads.

- **Docker Runtime Users:** Project-built images must create and run as a non-root UID/GID instead of relying on root defaults.
- **Pod Security Contexts:** Kubernetes manifests must set `runAsNonRoot`, explicit user and group IDs, `fsGroup` when writable volumes need group ownership, and `seccompProfile: RuntimeDefault`.
- **Container Security Contexts:** Containers must set `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, and `capabilities.drop: ["ALL"]` unless a documented workload exception requires otherwise.
- **Explicit Writable Paths:** Workloads with read-only root filesystems must model required write paths with PVCs or `emptyDir` volumes instead of reopening the root filesystem.
- **Service Account Boundaries:** Workloads that do not need Kubernetes API access should disable token automounting. Workloads that do need API access should keep scoped service accounts and RBAC.
- **Local Reproducibility:** Trivy is included in `shell.nix` so the same misconfiguration checks can be run from the development shell.

The first implementation applies this baseline to the chaos controller, sensor fleet, unified worker CronJobs, and n8n automation deployment.

### Rationale

- **Scanner findings become engineering standards:** Trivy findings are translated into source-controlled workload defaults rather than treated as one-off cleanup.
- **GitOps remains the source of truth:** ArgoCD reconciles hardened manifests from `k3s/`, so runtime drift does not become the control mechanism.
- **Read-only filesystems need explicit design:** n8n showed that hardening can expose hidden write paths. The correct fix is a targeted writable mount, not disabling the security control.
- **API access stays intentional:** The sensor fleet can disable service account tokens, while the chaos controller keeps a scoped token because it lists sensor pods.
- **Reviewers get repeatable evidence:** Local Trivy scans, Kustomize renders, and markdown documentation make the control easy to verify.

## Consequences

### Positive

- **Reduced container blast radius:** Workloads run without root defaults, unnecessary Linux capabilities, or mutable root filesystems.
- **Clearer security posture:** README and architecture docs now describe workload hardening as an implemented platform control.
- **Reproducible validation:** Developers can run Trivy from the Nix shell before pushing changes.
- **Better operational discipline:** Required write paths are visible in manifests as PVCs or `emptyDir` volumes.

### Negative

- **More manifest detail:** Security contexts and writable mounts add YAML verbosity.
- **Runtime compatibility checks are required:** Some third-party images, such as n8n, may need additional writable directories after `readOnlyRootFilesystem` is enabled.
- **Exceptions need documentation:** Any workload that cannot support the baseline must explain why and define a compensating control.

## Verification

- [x] **Kustomize Render:** `kubectl kustomize k3s/base/hardware-sim` completed successfully.
- [x] **Kustomize Render:** `kubectl kustomize k3s/base/worker` completed successfully.
- [x] **Kustomize Render:** `kubectl kustomize k3s/base/hub-apps` completed successfully.
- [x] **Trivy Docker Scan:** `nix-shell --run 'trivy config --severity HIGH,CRITICAL docker'` reported zero misconfigurations.
- [x] **Trivy K3s Scan:** `nix-shell --run 'trivy config --severity HIGH,CRITICAL k3s/base'` reported zero misconfigurations.
- [x] **Runtime Follow-Up:** n8n CrashLoopBackOff was traced to `/home/node/.cache` under a read-only root filesystem and fixed with an explicit `emptyDir` cache mount.
- [x] **Documentation:** README and security architecture docs were updated to describe the implemented workload hardening baseline.
