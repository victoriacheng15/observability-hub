# ADR 003: Shared Structured Logging Library

- **Status:** Accepted
- **Date:** 2026-01-03
- **Author:** Victoria Cheng

## Context and Problem Statement

As the project matures into a multi-service architecture (Proxy, System Metrics), logging has become fragmented.

- **Unstructured:** Plain text logs (`log.Printf`) are difficult to parse in Loki without fragile Regular Expressions.
- **Inconsistent:** Different services use different logging formats and field names.
- **Low Signal:** Metadata critical for debugging (e.g., service names, request duration, error levels) is often buried in unstructured strings.

## Decision Outcome

Implement a centralized logging strategy using a **Shared Go Package** pattern (`pkg/logger`), leveraging the standard library's `log/slog` (Go 1.21+).

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

## Consequences

### Positive

- **Observability:** Structured JSON logs are automatically parsed by Loki (no Regex needed).
- **Consistency:** Enforced by the library across all services.
- **Maintenance:** Centralized code in `pkg/logger` reduces duplication.

### Negative/Trade-offs

- **Rigidity:** Enforces a strict schema that might not fit every edge case.

## Verification

- [x] **Manual Check:** View logs in Grafana/Loki to confirm JSON structure.
