# Reading Sync Service Architecture

The Reading Sync Service (`services/reading-sync/`) is a scheduled data pipeline responsible for synchronizing cloud-based reading data into the local analytical store.

## 🧩 Component Details

- **Type**: Go service triggered via Systemd Timer.
- **Workflow**: Automated ETL (Extract, Transform, Load).
- **Storage**:
  - **Source**: MongoDB Atlas (External).
  - **Destination**: PostgreSQL (k3s).
- **Library Integration**: Uses the **Generic MongoStore** and **PostgresWrapper** from `pkg/db`.

## ⚙️ Logic Flow

1. **Trigger**: Systemd timer triggers the service (default: 00:00, 12:00).
2. **Extraction**: Connects to MongoDB via the Generic Wrapper and fetches documents with `status="ingested"`.
3. **Transformation**: Normalizes varying reading event schemas into a consistent `ReadingDocument` format.
4. **Loading**: Performs an idempotent insert into PostgreSQL `reading_analytics` using the Pure Wrapper.
5. **Confirmation**: Marks successfully processed documents in MongoDB as `status="processed"`.

## 🔭 Observability Implementation

Following the **Pure Wrapper** philosophy, the service implements high-fidelity telemetry:

- **Root Span**: `job.reading_sync` tracks the end-to-end lifecycle.
- **Advanced Metrics**:
  - `reading.sync.lag.seconds`: An **Observable Gauge** monitoring the time since the last successful execution.
  - `reading.sync.processed.total`: Counter for throughput.
- **Trace Attributes**: `db.documents.count` captures the batch size for each run.
- **Standardized Spans**: Automatically inherits `db.postgres.*` and `db.mongodb.*` spans from the core library.
