# Systemd Operations Cheat Sheet

This guide covers the essential commands for managing the Observability Hub's services and understanding their health status.

## 1. The "Big Picture" (Health Check)

Use these to see the health of the whole system at once.

| Command | Purpose |
| :--- | :--- |
| `systemctl list-units "openbao*" "proxy*" "reading*" "system-metrics*" "volume-backup*"` | **Daily Health Check.** Lists all project-related units. |
| `systemctl --failed` | **Critical.** Shows ONLY the services that have crashed/failed. |
| `systemctl list-timers` | Shows all active timers, when they last ran, and when they run next. |

---

## 2. Checking Specific Services

* **Check status with logs (The standard check):**

    ```bash
    systemctl status proxy.service
    ```

* **Check multiple services at once:**

    ```bash
    systemctl status proxy openbao tailscale-gate
    ```

---

## 3. Troubleshooting & Logs (`journalctl`)

Systemctl status only shows the last ~10 lines. Use `journalctl` to see the full history.

* **See last 50 lines (good for quick debug):**

    ```bash
    journalctl -u proxy.service -n 50 --no-pager
    ```

* **Follow logs in real-time (like `tail -f`):**

    ```bash
    journalctl -u proxy.service -f
    ```

* **See logs from a specific time:**

    ```bash
    journalctl -u reading-sync.service --since "1 hour ago"
    ```

---

## 4. Managing Services

* **Start/Stop/Restart:**

    ```bash
    systemctl start <service>
    systemctl stop <service>
    systemctl restart <service>
    ```

* **Reset a "Failed" status:**
    (Clears the red "failed" state without running the job again)

    ```bash
    systemctl reset-failed <service>
    ```

---

## 5. Critical Concept: Timers vs. Services

For scheduled jobs (like backups or metrics), you have two units:

1. **The Timer (`.timer`)**: The "Alarm Clock".
2. **The Service (`.service`)**: The "Worker".

### The "Inactive" Rule

It is **NORMAL** and **HEALTHY** for a "Oneshot" service to be `inactive (dead)` if it finished successfully.

| Unit Type | Healthy Status | Meaning |
| :--- | :--- | :--- |
| **Timer** | `active (waiting)` | The alarm is set. |
| **Service** | `inactive (dead)` | I finished my job and I'm sleeping. |
| **Service** | `failed` | **BAD.** The last job crashed. |

**Important:** A Timer can be `active` (green) even if the Service is `failed` (red). Always check `systemctl --failed` to catch broken jobs.
