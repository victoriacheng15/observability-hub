# System Metrics Collector Architecture

The System Metrics Collector (`system-metrics/`) is a lightweight agent designed to monitor the host infrastructure's health.

## Component Details

- **Runtime**: Go (compiled binary).
- **Library**: `gopsutil` for cross-platform hardware statistics.
- **Target**: Pushes data directly to the `system_metrics` table in PostgreSQL (TimescaleDB).

### Metrics Collected

- **CPU**: Usage percentage.
- **Memory**: Total, used, and available RAM.
- **Disk**: IOPS and usage stats.
- **Network**: Bandwidth usage (bytes sent/received).

## Data Flow: Metrics Collection

```mermaid
sequenceDiagram
    participant Host as Host Hardware
    participant Collector as System Metrics Collector
    participant DB as PostgreSQL
    participant Grafana as Grafana

    Host->>Collector: Sample stats (CPU, RAM, Disk)
    Collector->>Collector: Format as JSONB
    Collector->>DB: INSERT into system_metrics
    Grafana->>DB: Query time-series data
    DB-->>Grafana: Return metrics
```
