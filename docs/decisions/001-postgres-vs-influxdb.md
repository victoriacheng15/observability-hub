# RFC 001: PostgreSQL vs. InfluxDB for Metrics Storage

- **Status:** Accepted
- **Date:** 2025-11-01
- **Author:** Victoria Cheng

## The Problem

The observability system requires a storage engine to hold system metrics (CPU, Memory, Disk, Network) collected from the homelab.

Initially, **InfluxDB** was considered due to its specialized nature as a Time-Series Database (TSDB). However, this introduced architectural questions regarding long-term maintenance and data flexibility.

## Proposed Solution (PostgreSQL + JSONB)

Switch to **PostgreSQL** using the `JSONB` data type for metrics storage, with the option to use **TimescaleDB** if performance limits are reached.

### Rationale

- **Single Source of Truth:** PostgreSQL is already used for application data. Using it for metrics reduces the "Maintenance Tax" of managing multiple database engines.
- **SQL Proficiency:** SQL is a universal skill. InfluxQL/Flux is a niche language with limited transferability.
- **Flexibility:** JSONB allows storing unstructured telemetry data without strict schema migrations, which is ideal for an evolving observability project.
- **Operational Simplicity:** One backup strategy, one container to monitor, and one set of security patches to manage.

## Comparison

| Feature | InfluxDB | PostgreSQL (JSONB) |
| :--- | :--- | :--- |
| **Performance** | Optimized for high-write TS data. | Excellent for mid-range volume; expandable via TimescaleDB. |
| **Language** | InfluxQL / Flux (Niche). | SQL (Industry Standard). |
| **Ecosystem** | Strong within TIG stack. | Massive; works with every tool on earth. |
| **Maintenance** | High (Another DB to manage). | Low (Consolidated into existing DB). |

## Conclusion

While InfluxDB is a powerful tool for massive scale, the operational overhead and niche query language make it less ideal for a self-hosted homelab environment compared to the flexibility and ubiquity of PostgreSQL.
