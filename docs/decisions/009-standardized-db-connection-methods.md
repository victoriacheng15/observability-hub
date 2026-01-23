# RFC 009: Standardized Database Connection Methods

- **Status:** Accepted
- **Date:** 2026-01-21
- **Author:** Victoria Cheng

## The Problem

Currently, `pkg/db` only centralizes the *configuration* (DSN generation) of database connections. The actual *connection logic* (`sql.Open`, `mongo.Connect`, `Ping`) is duplicated across multiple services (`proxy`, `system-metrics`).

This leads to several issues:

1. **Code Duplication:** Every new service must rewrite the boilerplate for connecting and verifying (pinging) the database.
2. **Inconsistent Reliability:** Some services might strictly check `Ping()` on startup, while others might not. Retry logic and timeout configurations can drift.
3. **Dependency Scatter:** Managing driver versions (e.g., `lib/pq` vs `pgx`, `mongo-driver`) requires updating `go.mod` files in every single service directory.

## Proposed Solution

Expand the scope of `pkg/db` from a "Configuration Helper" to a "Connection Factory".

We will move the initialization logic into `pkg/db` by adding the following methods:

```go
package db

// ConnectPostgres establishes a connection to PostgreSQL and verifies it with a Ping.
// It returns a standard *sql.DB interface.
func ConnectPostgres(driverName string) (*sql.DB, error)

// ConnectMongo establishes a connection to MongoDB Atlas and verifies it with a Ping.
func ConnectMongo() (*mongo.Client, error)
```

### Dependency Management

This change effectively centralizes the database drivers into `pkg/db/go.mod`. Services will no longer need to direct import the drivers unless they need specific driver types. This simplifies dependency updates across the repo.

## Comparison / Alternatives Considered

| Feature | Current State (Decentralized) | Proposed State (Centralized Factory) |
| :--- | :--- | :--- |
| **Boilerplate** | High (Repeated in every `main.go`) | Low (One function call) |
| **Consistency** | Low (Varies by developer implementation) | High (Enforced by `pkg/db`) |
| **Dependency Mgmt** | Scattered (Update 5+ `go.mod` files) | Centralized (Update 1 `go.mod`) |
| **Module Size** | Lightweight | Heavier (Imports drivers) |

## Failure Modes (Operational Excellence)

- **Driver Bloat:** `pkg/db` will now pull in both Postgres and Mongo drivers. If a service only needs one, it still technically depends on the module that has both (though Go's build system is smart about dead code).
- **Versioning Conflicts:** If a service *needs* a specific different version of a driver than what `pkg/db` provides, it could cause `go.mod` conflicts. **Mitigation:** Enforce `pkg/db` as the single source of truth for driver versions.

## Conclusion

Centralizing connection logic aligns with our "Paved Road" philosophy. It reduces the cognitive load for creating new services and ensures that best practices (like Ping-on-startup) are applied universally.
