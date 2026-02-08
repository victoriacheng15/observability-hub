# ADR 013: The SRE Era - OpenTelemetry Integration

- **Status:** Accepted
- **Date:** 2026-02-07
- **Author:** Victoria Cheng

## Context and Problem Statement

This decision initiates the **SRE Era** of the platform—a phase dedicated to mastering Site Reliability Engineering principles through the implementation of industry-standard telemetry.

While the current custom Go collectors (e.g., `system-metrics`) and PostgreSQL storage are functional and effective for their original scope, they represent a "pre-standardized" phase of development. To evolve the platform into a true SRE learning hub, we need to bridge the gap between "working code" and "industry-standard observability."

The goal is not to "fix" the custom collectors, but to use the platform as a sandbox to understand:
- **Distributed Tracing**: How request flows are reconstructed across services.
- **OTel Specification**: Mastering the semantic conventions for metrics, logs, and traces.
- **Advanced Backend Management**: Operating specialized stores like Prometheus and Tempo.
- **FinOps & Sustainability**: Leveraging standardized metadata for granular cost and carbon analysis.

## Decision Outcome

Standardize the platform on **OpenTelemetry (OTel)** as the primary telemetry framework. This is a strategic shift into the "SRE Era," prioritizing the mastery of the OpenTelemetry ecosystem over ad-hoc collection methods.

### The Strategy

- **Deploy OpenTelemetry Collector**: Act as the central gateway for all telemetry signals within the k3s cluster.
- **Specialized Backends**: Move toward purpose-built storage engines:
  - **Prometheus**: For real-time operational metrics.
  - **Grafana Tempo**: For distributed trace storage.
  - **PostgreSQL**: Retained for long-term analytical and FinOps-specific data via OTel Collector aggregation.
- **Standardized Instrumentation**: All platform services will be updated to use OpenTelemetry SDKs (Go, Python, etc.) to emit telemetry.

### Rationale

- **Vendor-Agnostic**: OTel ensures the platform is not locked into a specific observability vendor.
- **Industry Standard**: Aligns the platform with modern SRE and DevOps practices, improving the "Developer Experience" and scalability.
- **Observability Trinity**: Completes the "Trinity" (Logs, Metrics, Traces) by adding distributed tracing capabilities.
- **FinOps Foundation**: Provides the standardized metadata (labels/attributes) required for granular resource and cost analysis.
- **Managed Resource Footprint**: By utilizing OTel's specialized backends and Kubernetes resource limits, we can maintain a lean cluster footprint while gaining advanced capabilities.

## Consequences

### Positive

- **Deep Visibility**: Enables distributed tracing to debug request flows and latency issues.
- **Consistent Data Model**: All telemetry follows a predictable structure across the entire fleet.
- **Future-Proofing**: Easy integration with future AI/ML or automation components (e.g., Kepler for energy tracking).
- **Reduced Database Bloat**: Moves high-frequency time-series data out of the relational database.

### Negative/Trade-offs

- **Infrastructure Complexity**: Adds 3-4 new components (Collector, Prometheus, Tempo, MinIO) to the cluster.
- **Operational Overhead**: Requires managing resource limits and retention policies for multiple specialized backends.
- **Refactoring Effort**: Requires updating existing Go services to use OpenTelemetry SDKs instead of direct DB writes for metrics.

## Verification

- [x] **Gateway Setup**: OpenTelemetry Collector running in k3s and verified accepting OTLP/HTTP payloads.
- [ ] **Storage Foundation**: MinIO and Grafana Tempo deployed and verified as persistent trace sinks.
- [ ] **Service Instrumentation**: `system-metrics` and `proxy` updated to emit telemetry via OTel SDKs.
- [ ] **Operationalization**: Prometheus scraping the Collector and Grafana visualizing metrics and traces from new sources.
