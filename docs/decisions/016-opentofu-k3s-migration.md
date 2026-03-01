# ADR 016: OpenTofu for Kubernetes Service Management

- **Status:** Accepted
- **Date:** 2026-02-25
- **Author:** Victoria Cheng

## Context and Problem Statement

Managing Kubernetes (K3s) Helm releases via `helm template → manifest.yaml → kubectl apply`
requires per-service make targets, generates committed manifest artifacts, and
provides no drift detection. As the number of services grows, this pattern
becomes error-prone and hard to audit.

Additionally, Terraform/OpenTofu is a core requirement for DevOps, SRE, and
Platform Engineering roles. This migration serves as a hands-on opportunity to
build practical IaC skills using OpenTofu, which is fully compatible with
Terraform's HCL syntax and provider ecosystem.

## Decision Outcome

Adopt OpenTofu to declaratively manage all standard Helm-based Kubernetes services.
Each service is defined as a `helm_release` resource referencing the existing
`k3s/<service>/values.yaml`. Existing live releases are migrated via
`tofu import` for zero-downtime adoption. MinIO serves as the S3-compatible
state backend. Collectors is explicitly excluded due to its custom local image
build and sideload workflow.

## Consequences

### Positive

- **Drift detection**: `tofu plan` surfaces any manual out-of-band changes.
- **Unified provisioning**: All standard services applied in one `tofu apply`.
- **No manifest artifacts**: Generated `manifest.yaml` files are eliminated from the repo.
- **Transferable skills**: HCL is the industry standard for IaC across all major clouds.

### Negative

- **Import overhead**: Each existing live release requires a `tofu import` step before the first apply.
- **Slower iteration**: `tofu apply` is slower than `helm upgrade` for rapid dev cycles.
- **State file risk**: State must be backed up; loss requires re-importing all resources.

## Verification

- [x] **Step 1 Complete:** OTel Collector running in observability, `tofu plan` shows zero diff.
- [x] **Step 2 Complete:** Grafana running in observability, dashboards load in browser.
- [x] **Step 3 Complete:** MinIO running in observability, Loki and Thanos buckets accessible.
- [x] **Step 4 Complete:** Prometheus running in observability, metrics visible in Grafana.
- [x] **Step 5 Complete:** Thanos running in observability, long-term metrics queryable via Grafana.
- [x] **Step 6 Complete:** Loki and Tempo running in observability, logs and traces visible in Grafana.
- [x] **Step 7 Complete:** PostgreSQL running in observability, application services reconnect successfully. (Note: OpenTofu apply may flag custom OCI tags like `:17.2.0-ext` as invalid, but state is synchronized via `helm upgrade` and `tofu import`).
