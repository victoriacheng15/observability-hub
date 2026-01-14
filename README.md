# Self-Hosted Observability Hub

A personal telemetry system that collects **system metrics** and **application events**, stores them in **PostgreSQL** (supercharged with **TimescaleDB** and **PostGIS**), and visualizes everything in **Grafana**, enabling high-performance time-series analysis and flexible data correlation.

---

## 🌐 Live Site

[Explore Live Telemetry & System Evolution](https://victoriacheng15.github.io/observability-hub/)

---

## 🎨 Design Philosophy

- **Holistic System Monitoring:** Tracks host health and application events through continuous telemetry, providing a unified view of system behavior over time.
- **Data Ownership & Self-Hosting:** Prioritizes self-hosted infrastructure for long-term data retention and full control, without cloud dependencies.
- **Reliability through Automation:** Employs simple, stateless services and a GitOps reconciliation engine to ensure a resilient, self-healing deployment that matches the declared state in Git.
- **Scale-Ready Storage:** Leverages PostgreSQL with TimescaleDB for efficient, long-term time-series analysis and flexible data correlation.

---

## 🛠️ Tech Stack

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![Grafana](https://img.shields.io/badge/grafana-%23F46800.svg?style=for-the-badge&logo=grafana&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

---

## 📚 Architectural Approach & Documentation

This section provides a deeper look into the system's structure, components, and data flow.

### System Architecture Diagram

This diagram shows the high-level flow of data from collection to visualization.

```mermaid
flowchart LR
    subgraph "External Sources"
        Apps
        Mongo(MongoDB Atlas)
    end

    subgraph "Internal Services"
        DS[Docker Services]
        SS[Systemd Services]
    end

    subgraph "Storage & Logs"
        PG(PostgreSQL)
        P(Promtail)
        L(Loki)
    end

    subgraph Visualization
        G(Grafana)
    end

    %% Data Pipeline
    Apps -- Events --> Mongo
    Mongo -- Pull --> DS
    DS -- Write --> PG
    SS -- Metrics --> PG
    PG --> G

    %% Logging Pipeline
    DS -- Logs --> P
    SS -- Logs --> P
    P --> L
    L --> G
```

### Component Breakdown

This table lists the main services and components within the observability hub, along with their responsibilities and location within the repository.

| Service / Component | Responsibility | Location |
| :------------------ | :------------- | :------- |
| **system-metrics** | A lightweight Go collector that gathers CPU, memory, disk, and network stats. | `system-metrics/` |
| **proxy** | A Go service acting as an API gateway and ETL engine for external data (e.g., MongoDB events). | `proxy/` |
| **page** | A Go static-site generator that builds the public-facing portfolio page. | `page/` |
| **PostgreSQL** | Primary time-series database for all metric and event data (with TimescaleDB and PostGIS extensions). | `docker-compose.yml` |
| **Grafana** | Primary visualization and dashboarding tool. | `docker-compose.yml` |
| **Loki** | Log aggregation system for all services. | `docker-compose.yml` |
| **Promtail** | Agent that ships host and container logs to Loki. | `docker-compose.yml` |
| **gitops-sync** | A `systemd` service that ensures the running state on the host matches the Git repository. | `systemd/` |
| **reading-sync** | A `systemd` service that periodically triggers the `proxy` ETL process via API. | `systemd/` |
| **Shared Libraries** | Reusable Go packages providing standardized logging, database connections, and common utilities. | `pkg/` |
| **Automation Scripts** | Collection of `scripts/` for maintenance, setup, and operational tasks. | `scripts/` |

### External Dependencies

These components exist outside this repository but are integral to the data pipeline:

| Dependency | Role |
| :--- | :--- |
| **Client Applications** | Sources of event data (e.g., Cover Craft, Personal Reading Analytics). |
| **MongoDB Atlas** | Interim cloud storage used as a buffer/queue for external event logs. |

### Data Flow

The system processes two main types of data: application events and host system metrics.

1. **Application Events (ETL Pipeline):**
    - **Source:** Client Applications (e.g., Cover Craft, Personal Reading Analytics Dashboard) write raw event logs to **MongoDB Atlas** (cloud).
    - **Ingestion:** The **reading-sync** service periodically triggers the **proxy** service.
    - **Transformation:** The **proxy** fetches new data from MongoDB, transforms it, and persists structured records into **PostgreSQL**.
2. **Host System Metrics:**
    - The **system-metrics** service (running on the host) collects CPU, Memory, Disk, and Network statistics.
    - Metrics are flushed directly to **PostgreSQL** at regular intervals.
3. **Visualization:**
    - **Grafana** queries **PostgreSQL** to visualize both application events and system health on unified dashboards.
4. **Logging:**
    - **Unified Aggregation:** **Promtail** tails logs from Docker containers (like `proxy`) and Systemd units (like `system-metrics`).
    - **Centralized Storage:** Logs are shipped to **Loki** and linked to metrics in Grafana for seamless correlation.

For deep dives into the system's inner workings:

- **[Detailed Architecture Docs](./docs/architecture/README.md)**: System context, component diagrams, and data flows.
- **[Decision Records](./docs/decisions/README.md)**: Architectural Decision Records (ADRs) explaining the "Why" behind key technical choices.

---

## 🚀 Getting Started (Local Development)

This guide will help you set up and run the `observability-hub` locally using Docker Compose.

### Prerequisites

Ensure you have the following installed on your system:

- [Go](https://go.dev/doc/install) (version 1.21 or newer)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (includes Docker Compose)
- `make` (GNU Make)

### 1. Configuration

The project uses a `.env` file to manage environment variables, especially for database connections and API keys.

```bash
# Start by copying the example file
cp .env.example .env
```

You will need to edit the newly created `.env` file to configure connections for MongoDB Atlas, PostgreSQL, and other services as required.

### 2. Build and Run the Stack

Bring up all services defined in `docker-compose.yml` with a single command:

```bash
make up
```

This command will:

- Build all necessary Docker images for the Go services.
- Start and provision all containerized services (PostgreSQL, Grafana, Loki, Promtail).
- The `-d` flag runs the services in the background.

To view the logs of all running services:

```bash
docker compose logs -f
```

### 3. Verification

Once the services are up and running, you can verify their functionality:

- **Grafana Dashboards:** Access Grafana at `http://localhost:3001`.
  - Default login: `admin` / `grafana`
  - You should see your provisioned data sources and dashboards.
- **Static Portfolio Site:** The `page` service builds your public portfolio site into the `page/dist` directory. You can inspect the generated static HTML files there.

### 4. Stopping the Stack

To stop and remove all running services and their associated containers, volumes, and networks:

```bash
make down
```
