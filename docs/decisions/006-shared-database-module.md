# ADR 006: Shared Database Configuration Module

- **Status:** Accepted
- **Date:** 2026-01-09
- **Author:** Victoria Cheng

## Context and Problem Statement

As the platform expands to include multiple services (`proxy`, `system-metrics`) connecting to the same PostgreSQL database, configuration logic has become duplicated and inconsistent.

- **Drift Risk:** Services maintain separate connection strings.
- **Code Duplication:** Environment variable parsing logic is repeated.
- **Timezone Ambiguity:** Some services were missing `timezone=UTC`.

## Decision Outcome

Extract the database connection configuration into a **Shared `pkg/db` Module**, following the "Paved Road" pattern.

### The "Single Source of Truth" Approach

A root-level module `pkg/db` will centralize how services connect to persistence layers. This module enforces "safe by default" configurations (e.g., `timezone=UTC`, `sslmode=disable`) and handles environment variable parsing.

## Consequences

### Positive

- **Consistency:** Connection logic is enforced by the library.
- **Maintenance:** Updates are centralized in `pkg/db`.
- **Reliability:** Defaults like `timezone=UTC` are applied universally.

### Negative

- **Dependency:** Services must rely on the shared module.

## Verification

- [x] **Manual Check:** Verify services connect successfully.
- [x] **Automated Tests:** `pkg/db` unit tests for DSN generation.
