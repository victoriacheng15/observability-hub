# RCA 006: Loki Gateway DNS Timeout

- **Status:** ✅ Resolved
- **Date:** 2026-04-01
- **Severity:** 🟡 Medium
- **Author:** Victoria Cheng

## Summary

Loki gateway access was degraded by nginx resolver timeouts inside the gateway. The gateway needed to resolve upstream Loki services, but its resolver configuration depended on the Kubernetes DNS name for `kube-dns` itself. That created a fragile circular dependency during gateway startup and query routing.

The permanent fix changed the Loki gateway nginx resolver to use the live `kube-dns` ClusterIP discovered by OpenTofu, avoiding DNS lookup of the DNS service name from inside the resolver path.

## Timeline

- **2026-04-01 23:48 UTC:** Fix merged in `25b78d4` to resolve Loki gateway DNS timeout with dynamic kube-dns lookup.
- **2026-04-01 23:48 UTC:** OpenTofu added a Kubernetes service data source for `kube-system/kube-dns`.
- **2026-04-01 23:48 UTC:** Loki gateway `nginxConfig.resolver` updated to use the discovered kube-dns ClusterIP directly.

## Root Cause Analysis

The primary cause was a **bootstrap dependency inside the Loki gateway resolver configuration**.

1. **Resolver depended on DNS**: The gateway was configured in a way that required resolving `kube-dns.kube-system.svc.cluster.local` before nginx could reliably resolve Loki upstreams.
2. **Gateway startup path was fragile**: If DNS resolution was slow or unavailable during startup, the gateway could timeout before it could route log queries or ingestion traffic.
3. **Static service names hid the dependency**: The manifest looked valid, but the runtime dependency chain was circular: the DNS resolver path itself needed DNS.

## Lessons Learned

- **Resolver configuration should avoid resolver names**: Components that configure their own resolver should use a stable IP or injected service address instead of relying on a DNS name for DNS itself.
- **OpenTofu can remove runtime guesswork**: Reading the Kubernetes service object at apply time provides a concrete ClusterIP and keeps generated Helm values aligned with the live cluster.
- **Gateway dependencies are production dependencies**: Even when Loki itself is healthy, a broken gateway can still create practical utility loss for logs.

## Action Items

- [x] **Fix:** Added `data.kubernetes_service_v1.kube_dns` in OpenTofu.
- [x] **Fix:** Set Loki gateway `nginxConfig.resolver` to the discovered kube-dns ClusterIP.
- [x] **Prevention:** Add a post-deploy Loki gateway query check that verifies gateway routing, not just Loki pod readiness.
- [x] **Process:** Document DNS bootstrap assumptions for components that manage their own resolver configuration.

## Verification

- [x] **Policy/Config Diff:** Verified `25b78d4` changed `tofu/05-observability.tf` to inject kube-dns ClusterIP into Loki gateway nginx config.
- [x] **Manual Check:** Confirm a log query through the Loki gateway succeeds after deployment.
- [x] **Automated Tests:** Add an observability smoke check that queries Loki through the same gateway path Grafana and agents use.
