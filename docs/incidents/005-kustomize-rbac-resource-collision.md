# RCA 005: Kustomize RBAC Resource Collision

- **Status:** ✅ Resolved
- **Date:** 2026-03-30
- **Severity:** 🟡 Medium
- **Author:** Victoria Cheng

## Summary

GitOps/Kustomize application of the K3s workload tree was destabilized by duplicate resources and namespace collisions introduced during the transition to tiered dev/prod overlays and shared RBAC definitions. The root kustomization mixed environment overlays with base resources, while some RBAC resources were namespace-sensitive but reused across namespaces.

The fix landed through a short sequence of commits that removed duplicate overlay inclusion, separated cluster-scoped RBAC from namespace-scoped resources, and reorganized the base/overlay hierarchy so dev and prod simulation resources could be rendered without colliding.

## Timeline

- **2026-03-31 01:22 UTC:** `664a2a8` restored RBAC manifest integrity and completed GPU bindings.
- **2026-03-31 01:33 UTC:** `64f9db6` removed dev/prod overlays from the root kustomization to fix duplicate resources.
- **2026-03-31 01:55 UTC:** `9e8d1aa` moved cross-namespace service accounts into `cluster-rbac.yaml` and reduced namespace collisions.
- **2026-03-31 16:31 UTC:** `b290eb1` refactored the Kustomize hierarchy for multi-namespace support.
- **2026-03-31 17:29 UTC:** `8c1bb0b` implemented tiered modular isolation and resolved remaining resource collisions.
- **2026-03-31 17:54 UTC:** `3a34e2e` standardized simulation environment variables and resolved remaining lint violations.

## Root Cause Analysis

The primary cause was **mixing environment overlays, base resources, and cross-namespace RBAC in the same render path**.

1. **Root kustomization included too much**: Including both overlays and base resources from the root caused the same objects to be rendered more than once.
2. **Namespace transformers affected shared resources**: ServiceAccounts, Roles, and RoleBindings that were intended for different namespaces were vulnerable to overlay namespace transformations.
3. **Cluster-scoped and namespace-scoped RBAC were not isolated**: GPU plugin and infrastructure identities needed explicit namespaces or cluster-level handling instead of being processed as generic base resources.
4. **Simulation overlays carried environment-specific patches**: Dev and prod simulation resources needed dedicated overlay handling for replica counts, namespaces, and target namespace environment variables.

## Lessons Learned

- **Base should be environment-neutral**: Root/base kustomizations should not include dev and prod overlays in the same render path.
- **RBAC needs scope boundaries**: Cluster-scoped and cross-namespace identities should live in files that cannot be accidentally namespace-transformed.
- **Render output is the contract**: Kustomize structure should be validated by rendering the final manifests for each overlay, not by inspecting YAML files independently.

## Action Items

- [x] **Fix:** Removed duplicate overlay inclusion from the root kustomization.
- [x] **Fix:** Split cross-namespace RBAC into `cluster-rbac.yaml`.
- [x] **Fix:** Refactored base directories to use dedicated kustomizations for hub apps, hardware simulation, and RBAC.
- [x] **Fix:** Standardized simulation `TARGET_NAMESPACE` configuration.
- [x] **Prevention:** Add CI checks that run `kubectl kustomize` or `kustomize build` for root, dev, and prod paths and fail on duplicate resource IDs.
- [x] **Process:** Document the rule that root kustomization may reference base resources and global policy, while environment overlays own environment-specific rendering.

## Verification

- [x] **Config Diff:** Verified the fix sequence from `664a2a8` through `3a34e2e` removed duplicate overlay paths, split RBAC scope, and restored valid simulation overlay rendering.
- [x] **Manual Check:** Render root, dev, and prod kustomizations and confirm there are no duplicate resource ID errors.
- [x] **Automated Tests:** Add Kustomize render validation to the lint or CI workflow.
