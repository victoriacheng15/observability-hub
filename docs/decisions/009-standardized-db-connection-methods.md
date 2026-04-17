# ADR 009: Standardized Database Connection Methods

- **Status:** Accepted
- **Date:** 2026-01-21
- **Author:** Victoria Cheng

## Context and Problem Statement

`internal/db` currently only centralizes configuration by returning the database connection string (DSN). The actual connection logic still lives inside individual services, with each service deciding how to call `sql.Open`, when to run `Ping`, how to handle startup failures, and which driver dependency to import.

This creates a weak abstraction: the DSN is shared, but the behavior around database connections is not. As more services depend on PostgreSQL and MongoDB, duplicated connection code increases the chance of inconsistent timeout handling, missing health checks, and scattered dependency updates.

The connection layer also needs to match the platform's internal-first architecture. Database setup is infrastructure behavior, not application business logic, so it belongs in one reusable package where lifecycle decisions, driver imports, and validation standards can be reviewed and tested once.

## Decision Outcome

Expand `internal/db` from a configuration helper into a **Connection Factory** that owns database initialization behavior.

- **Functionality:** `ConnectPostgres` and `ConnectMongo` handle initialization and verification (Ping).
- **Dependency Management:** Drivers are centralized in `internal/db/go.mod`.
- **Service Contract:** Callers receive a verified database handle instead of assembling connection behavior locally.
- **Boundary:** Services own queries and domain behavior; `internal/db` owns connection lifecycle and driver setup.

## Consequences

### Positive

- **Boilerplate Reduction:** One function call vs. 10+ lines of code per service.
- **Consistency:** Best practices (Ping on startup) applied universally.
- **Dependency Management:** Centralized driver updates.
- **Operational Reliability:** Startup failures become easier to diagnose because connection validation follows one shared path.
- **Testability:** Connection behavior can be covered in `internal/db` without duplicating test scaffolding across every service.

### Negative

- **Module Size:** `internal/db` pulls in multiple drivers (Postgres + Mongo).
- **Abstraction Scope:** Shared connection logic must stay generic enough to support multiple services without hiding service-specific query behavior.

## Verification

- [x] **Manual Check:** Verify services start without connection errors.
- [x] **Automated Tests:** `internal/db/postgres/client_test.go` and `internal/db/mongodb/client_test.go` verify initialization logic.
