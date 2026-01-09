# RFC 006: Shared Database Configuration Module

- **Status:** Accepted
- **Date:** 2026-01-09
- **Author:** Victoria Cheng

## The Problem

As the platform expands to include multiple services (`proxy`, `system-metrics`) connecting to the same PostgreSQL database, configuration logic has become duplicated and inconsistent.

- **Drift Risk:** Services maintain separate connection strings. Changing a default (e.g., enforcing `sslmode` or changing `timezone`) requires updates across multiple codebases.
- **Code Duplication:** Environment variable parsing logic (`DB_HOST`, `DB_PORT`, etc.) is repeated in every service.
- **Timezone Ambiguity:** Some services (like `system-metrics`) were missing the `timezone=UTC` flag, potentially leading to inconsistent data storage.

## Proposed Solution (Shared `pkg/db`)

Extract the database connection configuration into a shared `pkg/db` module, following the "Paved Road" pattern established by `pkg/logger`.

### The "Single Source of Truth" Approach

A root-level module `pkg/db` will centralize how services connect to persistence layers. This module enforces "safe by default" configurations (e.g., `timezone=UTC`, `sslmode=disable`) and handles environment variable parsing.

### Interface Design

The module provides a standardized function to generate the Data Source Name (DSN). It respects the `DATABASE_URL` environment variable if present, otherwise it constructs the DSN from individual components.

```go
package db

// GetPostgresDSN returns the formatted connection string based on environment variables.
// It prioritizes DATABASE_URL if set, otherwise it uses DB_HOST, DB_PORT, etc.
func GetPostgresDSN() (string, error)
```

## Comparison / Alternatives Considered

| Feature | Ad-Hoc Configuration (Old) | Shared `db` Module (Proposed) |
| :--- | :--- | :--- |
| **Consistency** | High risk of drift | Enforced by library |
| **Maintenance** | Manual updates in all services | Centralized in `pkg/db` |
| **Timezone** | Manual/Inconsistent | Forced `UTC` by default |
| **Simplicity** | Boilerplate in every `main.go` | Clean service initialization |

## Failure Modes (Operational Excellence)

- **Missing Configuration:** If required environment variables are missing, the module returns a descriptive error.
- **Dependency Bloat:** The module does not import SQL drivers, ensuring it remains lightweight and compatible with any driver (`lib/pq`, `pgx`, etc.) chosen by the service.

## Conclusion

Standardizing database configuration is a critical step towards architectural maturity. It ensures that all services interact with the persistence layer in a predictable, consistent, and observable manner.
