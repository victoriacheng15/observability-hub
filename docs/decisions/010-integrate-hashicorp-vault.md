# RFC 010: Integrate HashiCorp Vault

- **Status:** Proposed
- **Date:** 2026-01-22
- **Author:** Victoria Cheng

## The Problem

The current setup using static environment variables (`.env`) works perfectly for this personal repository. There is no technical debt or security failure driving this change.

However, relying solely on static configuration limits the opportunity to explore more advanced infrastructure patterns. By sticking to the simplest method, we miss the chance to experiment with **Centralized Secret Management**.

**Primary Motivation:**

1. **Exploration:** I want to understand how tools like HashiCorp Vault function architecturally (leasing, revocation, dynamic backends).
2. **Skill Expansion:** I want to implement a secure, centralized configuration flow to broaden my understanding of infrastructure components beyond just Docker and Systemd.
3. **Curiosity:** I am interested in seeing how much complexity—and capability—Vault adds to a simple Go application stack.

## Proposed Solution

We will integrate **HashiCorp Vault** as the centralized source of truth for all sensitive configuration.

### 1. Infrastructure

- **Vault Server:** Deployed via Docker Compose (`docker-compose.yml`) in "Dev Mode" for local simulation, utilizing a file-based or in-memory backend.
- **Initialization:** A seed script or manual initialization step will populate the Vault kv-store with the necessary secrets (`postgres_password`, `mongo_uri`, `webhook_secret`).

### 2. Application Integration

We will refactor the Go services (`proxy`, `system-metrics`) to use a tiered configuration strategy:

1. **Primary:** Attempt to fetch secrets from Vault via the official Go SDK.
2. **Fallback (Dev):** Fall back to environment variables if Vault is unreachable or unconfigured (optional, for local dev speed).

A new shared package `pkg/secrets` will handle the abstraction:

```go
type SecretStore interface {
    Get(key string) (string, error)
}

type VaultStore struct {
    client *api.Client
}
```

## Comparison / Alternatives Considered

| Option | Pros | Cons |
| :--- | :--- | :--- |
| **HashiCorp Vault** | Industry standard, supports dynamic secrets, platform agnostic. | High operational complexity. |
| **Kubernetes Secrets** | Native to K8s, simple API. | We are currently running a Hybrid (Docker/Systemd) setup, not full K8s. |
| **Cloud Provider (Azure Key Vault)** | Managed service, high availability. | Vendor lock-in; monthly cost; we want a self-hosted / cloud-agnostic "Hub". |
| **Status Quo (.env)** | Simple, works everywhere. | Insecure, no audit logs, no rotation. |

**Decision:** Vault provides the best opportunity to explore "Secret Management as a Service" and fits our self-hosted, cloud-agnostic philosophy.

## Failure Modes (Operational Excellence)

### 1. Vault Unavailability

- **Scenario:** The Vault container is down or sealed.
- **Impact:** Services (`proxy`, `metrics`) will fail to start (CrashLoopBackOff).
- **Mitigation:**
  - Implement robust retry logic with exponential backoff in the Go SDK wrapper.
  - (Optional) Local cache/fallback for non-critical secrets.

### 2. "Shared Fate"

- **Scenario:** Vault is the single point of failure.
- **Mitigation:** Clustered setups exist, but for this lab, we accept the risk and will monitor Vault health via `system-metrics`.

## Conclusion

Integrating HashiCorp Vault allows me to explore security patterns like secret rotation and centralized auditing within this repository. It turns a simple config task into a platform-level learning project.
