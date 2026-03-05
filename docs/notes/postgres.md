# PostgreSQL Note

## What You Should Run (Recommended Setup)

To initialize the database, you must first connect to the Postgres pod as the **superuser** (`postgres`). Run the following command from your host terminal:

```bash
kubectl exec -it postgres-postgresql-0 -n observability -- psql -U postgres
```

### 1. Application Owner (`server`)
The `server` user is for automated services (ingestion, proxy) that require write access to manage the platform's state.

```sql
-- Create dedicated database
CREATE DATABASE homelab;

-- Create application user
CREATE USER server WITH PASSWORD 'your_secure_password';

-- Make 'server' the owner of 'homelab' (best practice)
ALTER DATABASE homelab OWNER TO server;

-- Connect to the new database
\c homelab

-- Enable extensions (required per database)
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS postgis;
```

### 2. Agentic Investigator (`mcp_ro`)
The `mcp_ro` user is for the MCP-Domain server. It is strictly read-only to ensure AI agents can investigate without risking data integrity.

```sql
-- Create read-only investigator user
CREATE USER mcp_ro WITH PASSWORD 'agent_safe_password';

-- Connect to the target database
\c homelab

-- Grant connection and usage
GRANT CONNECT ON DATABASE homelab TO mcp_ro;
GRANT USAGE ON SCHEMA public TO mcp_ro;

-- Grant read-only access to current and future tables
GRANT SELECT ON ALL TABLES IN SCHEMA public TO mcp_ro;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO mcp_ro;
```

✅ **Resulting Permissions**:
- `server`: **Full ownership**. Can create, update, and delete all objects in `homelab`.
- `mcp_ro`: **Read-only**. Can perform `SELECT` and `EXPLAIN` but cannot modify state.

---

## How to Check Your Current Session (Inside `psql`)

- `SELECT current_user;` (shows your active role)
- `SELECT user;` (shorthand for current_user)
- `\conninfo` (user, database, and port)
- `\du` (list all roles/users)
- `\l` (list all databases)

💡 **Prompt Clues**:
- `postgres=#` (superuser)
- `server=>` (regular app user)
- `mcp_ro=>` (read-only agent user)

---

## Verification: Test Both Users

You can verify connectivity either from **inside the cluster** (via `kubectl`) or from the **host machine** (via the NodePort).

### Method A: From the Host (via NodePort)
This is how your host-based services (`proxy`, `mcp-telemetry`) will connect.

```bash
# Test 'server' (Write Access)
psql -h localhost -p 30432 -U server -d homelab

# Test 'mcp_ro' (Read-Only Access)
psql -h localhost -p 30432 -U mcp_ro -d homelab
```

### Method B: From the Cluster (via `kubectl`)
Use this for direct cluster-level debugging.

```bash
# Test 'server' (Write Access)
kubectl exec -it postgres-postgresql-0 -n observability -- psql -U server -d homelab

# Test 'mcp_ro' (Read-Only Access)
kubectl exec -it postgres-postgresql-0 -n observability -- psql -U mcp_ro -d homelab
```

---

## Testing Permissions

Once connected as `mcp_ro`, verify the security layer:

- `SELECT * FROM audit;` (Should succeed)
- `DROP TABLE audit;` (Should **FAIL** with permission denied)

✅ If the `DROP` command fails for `mcp_ro`, your security layer is correctly implemented!

> 🔒 **Security tip**: Store credentials in environment variables or a secrets manager; never hardcode!
