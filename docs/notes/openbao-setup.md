# OpenBao Setup & Operations Guide

## 1. Overview

We use **OpenBao** (an open-source fork of HashiCorp Vault) as our centralized secret store. It is installed via **Nix** and configured to store encrypted data locally on the filesystem.

- **Config:** `config/bao-local.hcl`
- **Data Storage:** `data/bao/` (Encrypted, git-ignored)
- **Port:** `8200` (UI available at `http://127.0.0.1:8200`)

---

## 2. Core Concepts

### 🔐 Initialization (Run ONCE)

- **What it does:** Generates the master encryption keys.
- **When to run:** Only when setting up a fresh instance (empty `data/bao` folder).
- **Output:** 5 Unseal Keys and 1 Root Token.
- **Status:** ✅ Completed on Jan 26, 2026.

### 🔓 Unsealing (Run Every Restart)

- **What it does:** Decrypts the master key in memory so the server can function.
- **When to run:** Every time the server restarts (reboot, crash, service restart).
- **Requirement:** You need **3 of the 5** unseal keys.
- **Status:** Required whenever `make bao-status` shows `Sealed: true`.

---

## 3. Operational Commands

**CRITICAL:** All `bao` commands must be run within the project's Nix environment and require the `BAO_ADDR` to be set.

### Recommended Workflow: Use Nix Shell

Enter the shell first to have all tools and variables ready:

```bash
nix-shell
export BAO_ADDR='http://127.0.0.1:8200'
```

### Unseal the Server (The "Unlock" Step)

If the server is running but sealed (e.g., after a reboot), follow these exact steps:

**Prerequisite:** Locate your 5 Unseal Keys (saved securely).

1. **Check Status:**

   ```bash
   make bao-status
   ```

   *If it says `Sealed: true`, continue.*

2. **Provide Keys (Repeat 3 times):**

   ```bash
   nix-shell --run "export BAO_ADDR='http://127.0.0.1:8200' && bao operator unseal"
   # Paste a different Unseal Key each time when prompted.
   ```

**Success Criteria:**
Run `make bao-status` again. It should now say:
> **Sealed: false**

### Log in (Root Access)

To add/read secrets, you need the Root Token.

```bash
export BAO_TOKEN=<Root Token>
nix-shell --run "export BAO_ADDR='http://127.0.0.1:8200' && bao token lookup"
```

---

## 4. Systemd Service (Production)

We use the project's **Makefile** to manage systemd services.

1. **Install & Start Service:**

   ```bash
   make install-services
   ```

   *This links `systemd/openbao.service` and enables it.*

2. **Check Logs:**

   ```bash
   journalctl -u openbao -f
   ```

**⚠️ Important:** Even with Systemd, the server will start in a **Sealed** state after a reboot. You must manually run the "Unseal" commands (Section 3) to make it operational.

---

## 5. Disaster Recovery (Lost Keys)

**If you lose the unseal keys, the data is unrecoverable.**
There is no "reset password" functionality. You must destroy the vault and start over.

**Steps to Reset (DATA LOSS):**

1. **Stop the Server:**

   ```bash
   sudo systemctl stop openbao
   ```

2. **Delete Encrypted Data:**

   ```bash
   rm -rf data/bao/*
   ```

3. **Restart Server:**

   ```bash
   sudo systemctl start openbao
   ```

4. **Re-Initialize:**

   ```bash
   nix-shell --run "export BAO_ADDR='http://127.0.0.1:8200' && bao operator init"
   ```

   *Save the NEW keys securely!*

5. **Re-Populate Secrets:**
   You must manually re-enter all secrets using `bao kv put`.

---

## 6. Troubleshooting

- **"Connection Refused":** The server is not running. Check logs: `journalctl -u openbao -e`.
- **"Vault is sealed":** The server is running but encrypted. Run the unseal commands.
- **"Permission Denied" (CLI):** You haven't exported `BAO_TOKEN` or `BAO_ADDR`.

---

## 🔐 Security Warning

**NEVER commit unseal keys or the root token to Git.**
If you lose the unseal keys, the data in `data/bao` is permanently lost.
