# Tailscale Gate Architecture

The Tailscale Gate (`scripts/tailscale_gate.sh`) is a security-focused monitoring script that manages the state of the public Tailscale Funnel based on the health of the Proxy service.

## 🎯 Objective

Ensure that the public entry point to the Observability Hub is only available when the backend processing service is healthy, minimizing the exposure of "dead" or unresponsive endpoints.

## 🧩 Component Details

- **Type**: Native Bash script.
- **Service**: `tailscale-gate.service` (Systemd).
- **Control Target**: `proxy.service`.

## ⚙️ Logic Flow

The script runs an infinite loop (default interval: 5 minutes) with the following logic:

1. **Service Check**: Executes `systemctl is-active --quiet proxy.service`.
2. **State: Healthy**:
   - Ensures `tailscale serve` is set to forward HTTPS (8443) to local HTTP (8085).
   - Ensures `tailscale funnel` is `ON`.
   - Logs: `Proxy RUNNING - Funnel OPEN`.
3. **State: Unhealthy**:
   - Turns `tailscale funnel` `OFF`.
   - Resets `tailscale serve` configuration.
   - Logs: `Proxy DOWN - Funnel CLOSED`.

## 🛰️ Networking Mapping

| Source (Public) | Gateway (Tailscale) | Target (Local) |
| :--- | :--- | :--- |
| `https://<node>.ts.net:8443` | Tailscale Funnel | `http://localhost:8085` |

## 🛡️ Security Impact

- **Automated Shield**: In the event of a Proxy crash or manual shutdown, the public funnel is automatically revoked within the next check cycle.
- **Service Decoupling**: The Gatekeeper is independent of the Proxy; it monitors the Proxy from the "outside" using standard Systemd status codes.
