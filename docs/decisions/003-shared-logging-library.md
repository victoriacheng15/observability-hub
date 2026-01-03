# RFC 003: Shared Structured Logging Library

**Status:** Accepted
**Date:** 2026-01-03
**Author:** Victoria Cheng

## The Problem

As the project matures into a multi-service architecture (Proxy, System Metrics), logging has become fragmented.

- **Unstructured:** Plain text logs (`log.Printf`) are difficult to parse in Loki without fragile Regular Expressions.
- **Inconsistent:** Different services use different logging formats and field names.
- **Low Signal:** Metadata critical for debugging (e.g., service names, request duration, error levels) is often buried in unstructured strings.

## Proposed Solution (Shared `pkg/logger`)

Implement a centralized logging strategy using a **Shared Go Package** pattern, leveraging the standard library's `log/slog` (Go 1.21+).

### The "Paved Road" Approach

A root-level module `pkg/logger` was created for all services to import. This module enforces a strict schema and configuration, removing the need for individual service developers to configure loggers manually.

### Standard Schema

All logs are output as JSON to `stdout`.

**Example:**

```json
{
  "time": "2026-01-01T12:00:00Z",
  "level": "INFO",
  "msg": "request_processed",
  "service": "proxy",
  "http_method": "GET"
}
```

**Core Fields:**

- `time`: RFC3339 Timestamp (automatic).
- `level`: Severity (INFO, WARN, ERROR, DEBUG).
- `msg`: The event description (snake_case).
- `service`: The name of the originating service (e.g., `proxy`).

## Implementation Details

The **Local Module Replacement** pattern was used to share code without publishing to a public repository.

- **Shared Module:** Created `pkg/logger` at the repository root.
- **Service Linkage:** Each service (`proxy`, `system-metrics`) uses a `replace` directive in its `go.mod` pointing to `../pkg/logger`.
- **Operational Excellence:** Implemented Docker-level log rotation via YAML anchors (`x-logging`) in `docker-compose.yml` to prevent disk exhaustion (10MB max, 3 files).
- **Ingestion:** Configured Promtail `pipeline_stages` to parse JSON logs and promote `service` and `level` to Loki labels.

## Comparison

| Feature | Ad-Hoc Logging (Old) | Shared `slog` Library (Implemented) |
| :--- | :--- | :--- |
| **Format** | Text / Unstructured | JSON / Structured |
| **Parsing** | Complex Regex in Loki | Automatic JSON Parsing |
| **Consistency** | Varies by developer | Enforced by library |
| **Maintenance** | Code duplicated in every service | Centralized in `pkg/logger` |

## Conclusion

Adopting a shared, structured logging library is a critical step towards **Platform Maturity**. It enables "Observability as Code" by ensuring that all telemetry entering our system is high-quality, queryable, and uniform by default.
