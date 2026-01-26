# ADR 010: Integrate OpenBao

- **Status:** Accepted
- **Date:** 2026-01-22
- **Author:** Victoria Cheng

## Context and Problem Statement

The current setup using static environment variables (`.env`) works perfectly for this personal repository. However, relying solely on static configuration limits the opportunity to explore **Centralized Secret Management** and learn industry-standard tools.

While HashiCorp Vault is the industry standard, its move to the Business Source License (BSL) limits its "open source" nature. **OpenBao**, a fork of Vault under the MPL 2.0 license, offers the same functionality while remaining truly open source and community-governed.

## Decision Outcome

Integrate **OpenBao** as the centralized source of truth for all sensitive configuration.

### Infrastructure

- **Server:** Run locally via `bao server` (installed via Nix).
- **Storage:** File-based persistence (initially).

### Application Integration

- **Tiered Config:** Go services try OpenBao first, fall back to `.env`.
- **Abstraction:** New `pkg/secrets` interface.
- **Compatibility:** Use existing Vault SDKs (API compatible).

## Consequences

### Positive

- **Open Source:** Strictly MPL 2.0 (no BSL concerns).
- **Learning:** Hands-on experience with secret rotation and leasing.
- **Security:** Moves away from static `.env` files (long-term).
- **Auditability:** OpenBao provides access logs for secrets.

### Negative/Trade-offs

- **Complexity:** High operational overhead compared to simple env vars.
- **Failure Mode:** Secret store becomes a single point of failure for app startup.

## Planned Verification

- [x] **Manual Check:** Verify services can fetch secrets from OpenBao.
- [x] **Automated Tests:** `pkg/secrets` unit tests with mocked client.
