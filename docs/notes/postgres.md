# PostgreSQL Note

## What You Should Run (Recommended Setup)

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

✅ This gives the `server` user:

- **Full ownership** of the `homelab` database
- Ability to create/modify **all objects** (tables, functions, etc.) inside `homelab`
- **No access** to other databases (e.g., `postgres`)
- **No superuser privileges** (secure by default)

> 💡 **Why this is better than `GRANT ALL PRIVILEGES`**:  
> Database ownership is cleaner, more maintainable, and avoids permission gaps.

---

## How to Check Your Current Session (Inside `psql`)

- `SELECT current_user;` → shows your role
- `SELECT user;` → shorthand for `current_user`
- `\conninfo` → full connection info (user, database, socket/port)
- `\du` → list all roles/users
- `\l` → list all databases

💡 **Prompt clues**:

- `postgres=#` → superuser
- `server=>` → regular user

---

## (Optional) If You *Don’t* Set Database Owner

If you prefer explicit grants (e.g., shared DB), run after `\c homelab`:

```sql
-- Grant schema usage
GRANT USAGE ON SCHEMA public TO server;

-- Allow creating new tables in public schema
GRANT CREATE ON SCHEMA public TO server;

-- Full access to existing objects
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO server;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO server;

-- Future-proof: auto-grant on new objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO server;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO server;
```

> ⚠️ **Note**: This is more complex and error-prone than database ownership.

---

## Test as the `server` User

From host terminal:

```sql
# Connect to your dedicated DB
docker exec -it postgres_server psql -U server -d homelab
```

Inside `psql`:

```sql
SELECT current_user;        -- should return 'server'
SELECT version();           -- should work
CREATE TABLE test (id SERIAL, name TEXT);
INSERT INTO test (name) VALUES ('hello');
SELECT * FROM test;
\dt                         -- should list 'test'
```

✅ If all commands succeed → your app user is ready for production!

> 🔒 **Security tip**: Store credentials in environment variables or a secrets manager; never hardcode!
