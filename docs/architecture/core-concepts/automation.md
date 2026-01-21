# Systemd Service Architecture

The Observability Hub leverages **Systemd** not just for process management, but as a core automation and reconciliation engine. By running lightweight agents and timers directly on the host, we ensure reliability independent of the Docker container runtime.

## Core Philosophy

- **Resilience through Decoupling**: Critical infrastructure (like GitOps and Security Gates) runs as native Systemd services to avoid "circular dependencies." This ensures the system can self-heal even if the Docker runtime is unresponsive.
- **Event-Driven Automation**: We prioritize webhooks over polling. By using the Proxy as an entry point, we trigger reconciliation only when changes actually occur, reducing CPU/Network overhead.
- **Unified Logging Standard**: All services (Go binaries and Bash scripts) emit JSON-formatted logs to `stdout`. This creates a high-fidelity observability pipeline managed by `journald` and Promtail.

## Service Inventory

The system consists of several main service families, each with a `.service` unit (the logic) and a `.timer` unit (the schedule).

| Service Name | Type | Schedule / Trigger | Responsibility |
| :--- | :--- | :--- | :--- |
| **`proxy`** | `simple` | Continuous | **API Gateway**: Core listener for data pipelines and GitOps webhooks. |
| **`tailscale-gate`** | `simple` | Continuous | **Security**: Monitors Proxy health and toggles Tailscale Funnel access. |
| **`gitops-sync`** | `oneshot` | **Webhook** | **Reconciliation**: Triggered by Proxy to pull latest code and apply changes. |
| **`reading-sync`** | `oneshot` | Daily (10:00 AM) | **Data Pipeline Trigger**: Calls Proxy API to sync MongoDB data to Postgres. |
| **`system-metrics`** | `oneshot` | Every 1 min | **Telemetry**: Collects host hardware stats and flushes them to the database. |

## Operational Excellence

Our systemd configurations employ several production-grade patterns:

- **Security Gating**: The `tailscale-gate` service implements a loop that ensures the public entry point (Funnel) is automatically closed if the underlying `proxy` service stops, preventing "dead" endpoints from being exposed.
- **Persistence (`Persistent=true`)**: Used in `reading-sync`. If the host is powered off during the scheduled time, systemd will trigger the service immediately upon the next boot.
- **Unified Logging**: All units output JSON-formatted logs to `stdout`, which are captured by `journald` and enriched with system metadata.

## Architectural Patterns

### 1. The "Webhook-Trigger" Pattern

For GitOps, we transitioned from polling (timers) to event-driven triggers. This reduces resource consumption and ensures faster deployment cycles.

```mermaid
sequenceDiagram
    participant GitHub
    participant Proxy as Go Proxy (Systemd)
    participant Script as gitops_sync.sh
    participant Journal as Journald

    GitHub->>Proxy: POST /api/webhook/gitops
    Proxy->>Proxy: Verify Signature & Event
    Proxy->>Script: Execute Background Task
    Script->>Journal: Log Sync Progress (JSON)
    Proxy-->>GitHub: 202 Accepted
```

### 2. Logging & Observability Integration

Unlike traditional "write to file" approaches, our systemd units write strictly to `stdout`/`stderr`. This creates a unified pipeline:

1. **Emission**: Service writes to `stdout`.
2. **Capture**: `journald` captures the stream and adds metadata (timestamp, unit name, PID).
3. **Collection**: Promtail (configured with `job_name: systemd-journal`) tails the journal.
4. **Ingestion**: Promtail pushes logs to Loki for visualization in Grafana.

## Configuration Structure

All unit files are stored in the `systemd/` directory of the repository and are deployed/updated by the `gitops-sync` script itself.

- **`[Unit]`**: Defines dependencies (e.g., `After=network.target`).
- **`[Service]`**: Defines the `ExecStart` command and `User=server`.
- **`[Install]`**: Defines the target (usually `multi-user.target`).
