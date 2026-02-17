# Proxy Service Architecture

The Proxy Service (`services/proxy/`) is a custom Go application that acts as the API gateway, Data Pipeline engine, and **GitOps automation trigger** for the platform. It runs as a native host process managed by Systemd.

## Component Details

### API Overview

| Endpoint | Method | Purpose |
| :--- | :--- | :--- |
| `/` | GET | Returns a JSON welcome message. |
| `/api/health` | GET | **Health Check**: Returns the service status and environment. |
| `/api/webhook/gitops` | POST | **GitOps Trigger**: Handles GitHub webhooks (Push/PR events) to sync local repositories. |
| `/api/sync/reading` | POST | Synchronizes reading data from MongoDB to PostgreSQL (TimescaleDB). |
| `/api/trace/synthetic/` | POST | **Synthetic Validation**: Ingests randomized metadata to stress-test the telemetry pipeline. |

### Endpoint Details

#### Health Check (`/api/health`)

Provides a simple endpoint for liveness and readiness probes.

- **Success**: Returns `200 OK` with basic service metadata.
- **Instrumentation**: Automatically traced via OpenTelemetry middleware.

#### GitOps Automation (`/api/webhook/gitops`)

This endpoint enables event-driven deployment.

1. **Verify**: Validates the `X-Hub-Signature-256` header using the `GITHUB_WEBHOOK_SECRET`.
2. **Filter**: Inspects the `X-GitHub-Event` and JSON payload to ensure the change is a **Push** to `main` or a **Merged PR** targeting `main`.
3. **Trigger**: Executes the local `scripts/gitops_sync.sh` script in the background.
4. **Log**: Broadcasts success/failure details to the system journal for observability.

#### Data Pipeline Engine (`/api/sync/reading`)

This endpoint triggers the extraction, transformation, and loading of data.

1. **Connect**: Establishes connection to MongoDB using `MONGO_URI`.
2. **Query**: Finds documents in the source collection where `status="ingested"`.
3. **Transform**: Converts documents into a standardized JSONB format.
4. **Load**: Inserts records into the PostgreSQL (TimescaleDB) `reading_analytics` table.
5. **Update**: Marks the original MongoDB documents as `status="processed"`.

## Distributed Tracing

The Proxy Service is instrumented with the **OpenTelemetry SDK** to provide visibility into request lifecycles and pipeline performance.

### Configuration

The service initializes a global TracerProvider during startup, controlled by environment variables:

| Variable | Description |
| :--- | :--- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | The gRPC endpoint of OpenTelemetry (e.g., `localhost:30317`). |
| `OTEL_SERVICE_NAME` | The service identifier used in traces (defaults to `proxy`). |

### Trace Coverage

Spans are manually or automatically created for:

- **GitOps Ingestion**: Tracking webhook validation and script execution time.
- **Data Pipeline (ETL)**: Measuring MongoDB fetch latency and PostgreSQL insertion throughput.
- **Synthetic Validation**: Testing pipeline fidelity with randomized business metadata.
- **API Requests**: Correlating incoming requests with backend pipeline activities.

Traces are exported to the central **OpenTelemetry Collector** via gRPC and stored in **Grafana Tempo**.

### Instrumentation Strategy

1. **Automatic HTTP Tracing**: The service uses `otelhttp` middleware to wrap the primary mux, automatically capturing spans for every incoming request, including path, method, and status codes.
2. **Manual Spans**: High-value logic (signature verification, database transactions, script execution) is instrumented with manual spans to provide granular timing and error context within the request lifecycle.
