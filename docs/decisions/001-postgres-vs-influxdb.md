# ADR 001: PostgreSQL vs. InfluxDB for Metrics Storage

- **Status:** Accepted
- **Date:** 2025-11-01
- **Author:** Victoria Cheng

## Context and Problem Statement

The observability system requires a storage engine to hold system metrics (CPU, Memory, Disk, Network) collected from the homelab.

Initially, **InfluxDB** was considered due to its specialized nature as a Time-Series Database (TSDB). However, this introduced architectural questions regarding long-term maintenance and data flexibility.

## Decision Outcome

Switch to **PostgreSQL** using the `JSONB` data type for metrics storage, with the option to use **TimescaleDB** if performance limits are reached.

### Rationale

- **Single Source of Truth:** PostgreSQL is already used for application data. Using it for metrics reduces the "Maintenance Tax" of managing multiple database engines.
- **SQL Proficiency:** SQL is a universal skill. InfluxQL/Flux is a niche language with limited transferability.
- **Flexibility:** JSONB allows storing unstructured telemetry data without strict schema migrations, which is ideal for an evolving observability project.
- **Operational Simplicity:** One backup strategy, one container to monitor, and one set of security patches to manage.

## Consequences

### Positive

- **Ecosystem:** Massive ecosystem; works with every tool on earth.
- **Maintenance:** Low operational overhead (consolidated into existing DB).
- **Language:** Uses SQL (Industry Standard) instead of InfluxQL/Flux (Niche).

### Negative

- **Performance:** PostgreSQL is excellent for mid-range volume but less optimized for high-write TS data compared to InfluxDB (though expandable via TimescaleDB).

## Verification

- [x] **Architecture Check:** Confirm `docker-compose.yml` runs a `postgres` service and **does not** run `influxdb`.
- [x] **Code Check:** Verify `go.mod` depends on `lib/pq` (Postgres driver) and not `influxdb-client-go`.
