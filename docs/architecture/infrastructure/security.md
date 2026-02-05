# Security Architecture

The Observability Hub employs a multi-layered security model to protect the data pipeline while maintaining external accessibility for webhooks.

## 📡 External Ingress (Tailscale Funnel)

To receive webhooks from GitHub without exposing the entire server to the public internet, we use **Tailscale Funnel**.

- **Scoped Exposure**: Only port `8443` (HTTPS) is exposed to the public.
- **Termination**: TLS is terminated at the Tailscale edge; traffic is forwarded to the local Proxy service over the secure Tailscale mesh.
- **Dynamic Gating**: The `tailscale-gate` service ensures the funnel is closed if the Proxy is inactive.

## 🔐 Webhook Authentication

All incoming requests to `/api/webhook/gitops` must be authenticated using **HMAC-SHA256 signature verification**.

1. **Secret Storage**: The `GITHUB_WEBHOOK_SECRET` is stored in the `.env` file (not committed to Git).
2. **Verification**: The Proxy validates the `X-Hub-Signature-256` header against the request body before any processing occurs.
3. **Event Filtering**: Only specifically defined events (`push` or `pull_request` to `main`) are processed; all others are rejected early to minimize resource consumption.

## 🧪 Hybrid Isolation

- **Kubernetes Isolation**: The data tier (PostgreSQL, Loki, Grafana) runs inside the **`observability` namespace** within k3s. They are isolated from the host and other namespaces via Kubernetes NetworkPolicies and are not accessible from the public internet.
- **Service Boundaries**: The Proxy (running as a native systemd service) acts as the primary "bridge" between the public funnel and the internal cluster data tier.
- **Environment Variables**: Sensitive credentials (passwords, URIs) are managed via **Kubernetes Secrets** or retrieved from OpenBao, ensuring they never appear in plain text in process lists.

## 🧱 Repository Integrity

The GitOps agent (`gitops_sync.sh`) uses **Fast-Forward Only (`--ff-only`)** merges to prevent accidental merge commits on the host and ensure the local environment stays strictly in sync with the remote "Source of Truth."
