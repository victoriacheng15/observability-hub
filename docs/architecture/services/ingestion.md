# Ingestion Service Architecture

The Ingestion Service (`cmd/ingestion/`) is a unified data orchestration engine responsible for synchronizing external data sources into the platform's local analytical store (PostgreSQL). It operates as a periodic task runner managed by a Systemd timer.

## Component Details

### Task Overview

The service follows a **Task-Oriented Design**, where specific data synchronization logics are encapsulated into independent, testable tasks managed by a centralized engine.

| Task | Source | Purpose |
| :--- | :--- | :--- |
| `reading` | MongoDB Atlas | **Reading Analytics**: Syncs article metadata and engagement metrics from cloud to local store. |
| `brain` | GitHub Issues | **Second Brain**: Ingests journal entries, performs atomization, and calculates token counts. |

### Logic Details

#### Reading Analytics Task (`reading`)

Synchronizes Cloud-based MongoDB data with the local PostgreSQL environment.

1. **Fetch**: Retrieves unprocessed documents from MongoDB Atlas in configurable batches.
2. **Transform**: Maps MongoDB BSON/JSON metadata to the structured PostgreSQL `reading_analytics` schema.
3. **Persist**: Executes UPSERT operations in PostgreSQL to ensure data consistency.
4. **Acknowledge**: Marks documents as "processed" in MongoDB to prevent duplicate ingestion.

#### Second Brain Task (`brain`)

Transforms GitHub-based journaling entries into atomic, searchable database records.

1. **Check**: Queries PostgreSQL for the most recent entry date to determine the sync delta.
2. **Ingest**: Fetches new journal entries from the configured GitHub repository via the GitHub API.
3. **Atomize**: Decomposes long-form markdown logs into granular "thought atoms," including metadata like tags and categories.
4. **Quantify**: Calculates token counts for each atom to support future LLM-based analytical workloads.

## Distributed Tracing

The Ingestion Service is instrumented with the **OpenTelemetry SDK** to provide visibility into the data pipeline performance and task status.

### Configuration

The service initializes a global TracerProvider during startup, controlled by environment variables:

| Variable | Description |
| :--- | :--- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | The gRPC endpoint of OpenTelemetry (e.g., `localhost:30317`). |
| `OTEL_SERVICE_NAME` | The service identifier used in traces (defaults to `ingestion`). |

### Trace Coverage

Spans are created for the entire job lifecycle:

- **Job Lifecycle**: Root span `job.ingestion` tracks the overall synchronization run.
- **Task Execution**: Child spans (`task.reading`, `task.brain`) provide granular visibility into individual task performance.
- **API/DB Operations**: Sub-spans for GitHub API requests, MongoDB fetches, and PostgreSQL transactions.

Traces are exported to the central **OpenTelemetry Collector** via gRPC and stored in **Grafana Tempo**.

### Instrumentation Strategy

1. **Task Engine Wrapper**: The service uses a centralized `RunTask` engine that automatically wraps every registered task in a named OpenTelemetry span, capturing task-specific attributes and success/failure status.
2. **Manual Spans**: High-latency operations (external API calls and complex database syncs) are manually instrumented to provide precise timing and error context for pipeline optimization.
