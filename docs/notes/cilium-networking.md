# Cilium & Hubble: Network Layer Intelligence

This note documents how the Observability Hub currently uses Cilium and Hubble for network visibility, service isolation, and operational troubleshooting.

## What Cilium Is Doing Here

Cilium replaces the default K3s networking datapath with an eBPF-based model.

In this repo, that gives us three practical capabilities:

- service-to-service identity at L3/L4
- selective HTTP and gRPC visibility at L7
- clusterwide policy enforcement without sidecars

Hubble is the operator-facing view into that datapath. It turns raw flows into verdicts, labels, ports, and protocol metadata that can be inspected in the UI or scraped as metrics.

## Reading Hubble Output

Hubble is most useful when read as a sequence of questions.

| Layer | Question | Example |
| :--- | :--- | :--- |
| L3 | Who is talking? | `grafana` to `prometheus` |
| L4 | How is it talking? | TCP `9090`, verdict `FORWARDED` |
| L7 | What is it doing? | `GET /api/v1/query` |

Typical interpretations:

- `FORWARDED`: the flow matched an allowed path
- `DROPPED`: a policy blocked the flow or a required allow rule is missing
- L7 errors with allowed L4 traffic: the network path is open, but the application is failing upstream

For dropped traffic, check Hubble first, then confirm with pod logs and service health.

## Current Policy Layout

The active policy model in this repo is built around Cilium clusterwide policies.

### L7 Visibility Layer

The `observability-l7` policy enables HTTP and gRPC inspection for selected services:

- OpenTelemetry Collector
- Loki
- Tempo
- Prometheus
- MinIO
- Thanos
- Kepler

This is visibility-oriented policy, not full application isolation. It tells Cilium which ports are worth decoding at L7.

### Core Observability Layer

The `observability-core` policy protects and enables shared platform traffic for:

- OpenTelemetry Collector
- Loki
- Tempo
- Prometheus
- Thanos
- Kepler
- kube-state-metrics
- EMQX
- prometheus-node-exporter

It allows cluster traffic on the known observability ports, plus DNS and Kubernetes API access needed for basic operation.

### Namespace-Specific Layers

There are separate clusterwide policies for:

- `databases`
- `argocd`
- `hub`

These policies provide a namespace boundary, but they are not equally strict.

## Current Security Posture

The cluster has useful segmentation, but it is not full zero-trust yet.

Important exceptions:

- `hub` allows wildcard FQDN egress for n8n and related external API traffic
- `databases` allows wildcard FQDN egress for pgAdmin and related admin traffic
- several UI and admin ports still allow ingress from `world`

This is intentional for now. It keeps core workflows working while the actual dependency graph is still being documented.

ArgoCD is the notable exception: its git egress is already scoped to GitHub and GitLab patterns instead of wildcard external access.

## L7 Coverage in Practice

The current L7 ports matter because they map directly to platform behavior:

- `3100`: Loki HTTP ingestion and query paths
- `4318`: OTLP over HTTP
- `4317`: OTLP over gRPC
- `3200`: Tempo HTTP
- `9090`: Prometheus HTTP
- `9000`: MinIO S3 API
- `10901` and `10902`: Thanos gRPC and HTTP
- `28282`: Kepler metrics endpoint

If Hubble is not showing L7 details for one of these services, verify both of the following:

- the service label matches the `observability-l7` selector
- the relevant port is included in the L7 policy

## Operational Constraints

This cluster is single-node and already had a Cilium recovery incident. That changes how policy changes should be made.

Practical implications:

- prefer reversible, additive changes over broad lockdowns
- treat DNS and Kubernetes API reachability as first-class validation targets
- avoid treating `kube-system` as the next easy hardening target
- do not assume a policy change is low risk just because it looks narrow in YAML

Policy regressions can break critical paths quickly on a single-node control-plane system.

## Metrics and Alerting

Hubble metrics are already scraped by Prometheus and available for dashboards.

That supports:

- drop-rate monitoring
- traffic inspection dashboards
- troubleshooting of missing allows and noisy denies

Alerting is a separate concern. Hubble metrics existing does not mean policy-drop alerting is already wired end to end. If alerting is added later, verify the delivery path first.

## Safe Workflow for Policy Changes

Before tightening a policy:

1. Document the expected traffic path first.
2. Verify whether the flow is namespace-local, cross-namespace, or external.
3. Confirm DNS, Kubernetes API, and storage dependencies.
4. Apply the narrowest rule that preserves the known workflow.
5. Watch Hubble for dropped flows immediately after rollout.

Minimum validation checklist:

- Grafana can query Prometheus, Loki, and Tempo
- n8n can reach Postgres and any required external APIs
- ArgoCD can resolve DNS, reach the Kubernetes API, and sync repositories
- Tempo, Loki, and Thanos can reach MinIO
- pgAdmin remains reachable for required admin workflows

## What To Document Next

The highest-value documentation gap is not more theory. It is the actual allowed flow baseline.

That baseline now lives in `docs/notes/network-flow-baseline.md`.

It should continue to cover:

- `observability` to `databases`
- `hub` to `observability`
- `hub` to `databases`
- `argocd` to Kubernetes API and git remotes
- external destinations currently needed by n8n and pgAdmin

Once those paths are explicit, targeted hardening becomes realistic and much safer.
