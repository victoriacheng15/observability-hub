# Self-Hosted Observability Hub

A personal telemetry system that collects **system metrics** and **application events**, stores them in **PostgreSQL** (supercharged with **TimescaleDB** and **PostGIS**), and visualizes everything in **Grafana**, enabling high-performance time-series analysis and flexible data correlation.

---

## 🌐 Live Site

[Explore Live Telemetry & System Evolution](https://victoriacheng15.github.io/observability-hub/)

---

## 🎨 Design Philosophy

- **Continuous Contextual Telemetry:** Built to understand system behavior over time through continuous telemetry rather than one-off checks.
- **Systemic Health Monitoring:** Tracks host health, debugs pipeline failures, and connects infrastructure metrics to application events.
- **Data Ownership:** Prioritizes 100% self-hosted infrastructure and long-term data retention without cloud dependencies.
- **Simplicity & Reliability:** Employs scheduled, stateless, and idempotent Go services over complex, heavy-weight agents.
- **Scale-Ready Storage:** Leverages TimescaleDB (PostgreSQL) for efficient time-series analysis and historical tracking.

---

## 📚 Architectural Approach & Documentation

| Component | Approach |
| :--- | :--- |
| **Collectors** | Custom Go services—cron-driven, stateless, and idempotent. |
| **Storage** | **TimescaleDB** with JSONB for unified metric/event storage. |
| **Visualization** | **Grafana** dashboards separated by concern (infra vs. app). |
| **Observability** | **Loki** (logs) and a **Go proxy** (ETL) for full telemetry coverage. |

For deep dives into the system's inner workings:

- **[Architecture](./docs/architecture/README.md)**: System context, component diagrams, and data flows.
- **[Decisions](./docs/decisions/README.md)**: Architectural Decision Records (ADRs) explaining the "Why."

---

## 🛠️ Tech Stack

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![Grafana](https://img.shields.io/badge/grafana-%23F46800.svg?style=for-the-badge&logo=grafana&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

---

## 🔍 What It Does

| Capability | Details |
| :--- | :--- |
| **System Metrics** | Collects CPU, memory, disk, and network stats via a lightweight Go collector (`gopsutil`). |
| **App Events** | Tracks application events from [personal reading analytics dashboard](https://github.com/victoriacheng15/personal-reading-analytics-dashboard). |
| **Storage** | Stores data in **TimescaleDB (PostgreSQL)** using a flexible JSONB schema for cross-type querying. |
| **Visualization** | **Grafana** dashboards for **Infrastructure Health** (metrics) and **Application Telemetry** (trends). |
| **Reliability** | Ensures durability via automated volume backups and idempotent collectors. |

---

## 🚀 Explore the Live Site

[Explore Live Telemetry & System Evolution](https://victoriacheng15.github.io/observability-hub/)
