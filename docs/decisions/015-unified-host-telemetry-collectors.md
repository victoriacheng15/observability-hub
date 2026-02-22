# ADR 015: Unified Host Telemetry Collectors

- **Status:** Accepted
- **Date:** 2026-02-21
- **Author:** Victoria Cheng

## Context and Problem Statement

The Observability Hub currently utilizes a fragmented approach for host-level telemetry:

- **Grafana Alloy:** A standalone agent scraping Tailscale logs via `journalctl`. Despite its minimal workload, its actual idle consumption was approximately 10m CPU and 50Mi RAM, but it required significant reserved resources (Requests: 20m CPU / 114Mi RAM).
- **Existing `system-metrics`:** A Go service collecting host stats via `gopsutil` every minute, leading to constant database writes and unnecessary resource overhead.
- **`systemd` units:** Managing legacy collection scripts on the host adds operational complexity.

This fragmentation results in a high "reservation tax" where ~60MiB of RAM is guaranteed but unused, and architectural inconsistency where core monitoring logic is split across multiple agents and scripts.

## Decision Outcome

Consolidate all host-level observability responsibilities into a single, re-architected **Host Telemetry Collectors Service (`collectors`)** deployed as a Kubernetes DaemonSet.

### Key Architectural Shifts

- **Thanos-Centric Metrics:** Host metric collection (CPU, RAM, Disk, Network, Temperature) is now retrieved from **Prometheus** (exposed via Thanos Query). This leverages the unified API for both real-time and long-term storage (MinIO).
- **Batch Processing Model:** Move from 1-minute continuous polling to a **15-minute batch interval** (as a starting point). The service wakes up every 15 minutes, performs a range query with `step=1m` to maintain granularity, and batch-inserts results into PostgreSQL.
- **Unified Tailscale Collection:** Incorporate Tailscale status and log collection (via `exec.Command`) directly into the Go service, exposing them via OpenTelemetry and PostgreSQL.
- **Resource Optimization:** Configure the new service with tight resource requests (10m CPU / 40Mi RAM), releasing significant guaranteed memory back to the cluster.

### Rationale

- **Efficiency of Batch Processing:** Research confirms that moving from a continuous 1-minute polling cycle to a 15-minute batch interval significantly reduces the average CPU duty cycle. The service transitions from a constant baseline draw to a "wake-perform-sleep" model, making it virtually invisible to the CPU scheduler for 99% of its operational life.
- **Optimization of Reserved Resources:** Empirical observation via `kubectl top` reveals a significant reduction in both idle and reserved resource allocation. The specialized Go service for Collectors now consumes ~2m CPU / 10Mi RAM when idle, with requests set to 10m CPU / 40Mi RAM, effectively returning significant guaranteed CPU and memory resources back to the cluster nodes.

  | Component              | Idle CPU / RAM | Reserved CPU / RAM |
  | :--------------------- | :------------- | :----------------- |
  | **Legacy Agent (Alloy)** | ~10m / 50Mi    | 20m / 114Mi        |
  | **Unified Collector**  | ~2m / 10Mi     | 10m / 40Mi         |
  | **Net Savings**        | **~8m / 40Mi** | **10m / 74Mi**     |
  | **% Reduction**        | **80% / 80%**  | **50% / 65%**      |

- **Data Parity & Schema Consistency:** By utilizing PromQL `query_range` with a `step=1m`, we maintain the high-resolution data (1-minute granularity) required for accurate FinOps analysis while gaining the operational benefits of batch processing.
- **Surgical Consolidation:** This approach allows us to integrate specialized collection (Tailscale, hardware temperatures) into a single path, eliminating the need for three separate management domains (Alloy, systemd, and legacy Go services).

## Consequences

### Positive

- **Significant Resource Savings**: Frees up approximately 8m CPU and 74Mi RAM in reserved resources per node, based on the difference between Alloy's prior requests (20m CPU / 114Mi RAM) and Collectors' new requests (10m CPU / 40Mi RAM), with even larger savings in actual idle usage.
- **Operational Simplicity**: Replaces three legacy components (Alloy, old `system-metrics`, `systemd` units) with one unified Go binary.
- **FinOps Readiness**: Provides a curated, efficient historical data source in PostgreSQL for electricity cost analysis.
- **Architectural Alignment**: Standardizes on Go and the "library-first" pattern.

### Negative

- **Increased Development Effort**: Requires custom Go code for Tailscale and PromQL parsing rather than using off-the-shelf Alloy modules.
- **Dependency on Thanos**: Data collection for FinOps now depends on the availability of the Thanos Query service.

## Verification

- [x] **Resource Usage:** Monitor `collectors` pod via `kubectl top` and ensure it operates within the new 40Mi/80Mi RAM limits.
- [x] **Data Parity:** Confirm `system_metrics` table in PostgreSQL receives 1-minute interval data for all four metric types plus hardware temperature.
- [x] **Tailscale Flow:** Verify `tailscale_*` logs appear in Grafana (via Loki) and `collectors.tailscale.active` metrics appear in Grafana (via Prometheus/OTel).
- [x] **Decommissioning:** Confirm `alloy` and legacy `system-metrics` units are stopped and removed.
