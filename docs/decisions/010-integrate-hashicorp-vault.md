# ADR 010: Integrate HashiCorp Vault

- **Status:** Proposed
- **Date:** 2026-01-22
- **Author:** Victoria Cheng

## Context and Problem Statement

The current setup using static environment variables (`.env`) works perfectly for this personal repository. However, relying solely on static configuration limits the opportunity to explore **Centralized Secret Management** and learn industry-standard tools like HashiCorp Vault.

## Decision Outcome

Integrate **HashiCorp Vault** as the centralized source of truth for all sensitive configuration.

### Infrastructure

- **Vault Server:** Deployed via Docker Compose in "Dev Mode".
- **Initialization:** Seeded with secrets (`postgres_password`, etc.).

### Application Integration

- **Tiered Config:** Go services try Vault first, fall back to `.env`.
- **Abstraction:** New `pkg/secrets` interface.

## Consequences

### Positive

- **Learning:** Hands-on experience with secret rotation and leasing.
- **Security:** Moves away from static `.env` files (long-term).
- **Auditability:** Vault provides access logs for secrets.

### Negative/Trade-offs

- **Complexity:** High operational overhead compared to simple env vars.
- **Failure Mode:** Vault becomes a single point of failure for app startup.

## Planned Verification

- [ ] **Manual Check:** Verify services can fetch secrets from Vault.
- [ ] **Automated Tests:** `pkg/secrets` unit tests with mocked Vault client.
