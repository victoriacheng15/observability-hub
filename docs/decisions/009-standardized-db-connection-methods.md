# ADR 009: Standardized Database Connection Methods

- **Status:** Accepted
- **Date:** 2026-01-21
- **Author:** Victoria Cheng

## Context and Problem Statement

`internal/db` currently only centralizes configuration (DSN). The actual connection logic (`sql.Open`, `Ping`) is duplicated across services, leading to inconsistent reliability and scattered dependency management.

## Decision Outcome

Expand `internal/db` to be a **Connection Factory**.

- **Functionality:** `ConnectPostgres` and `ConnectMongo` handle initialization and verification (Ping).
- **Dependency Management:** Drivers are centralized in `internal/db/go.mod`.

## Consequences

### Positive

- **Boilerplate Reduction:** One function call vs. 10+ lines of code per service.
- **Consistency:** Best practices (Ping on startup) applied universally.
- **Dependency Management:** Centralized driver updates.

### Negative

- **Module Size:** `internal/db` pulls in multiple drivers (Postgres + Mongo).

## Verification

- [x] **Manual Check:** Verify services start without connection errors.
- [x] **Automated Tests:** `proxy/utils/dbConnection_test.go` verifies initialization logic.
