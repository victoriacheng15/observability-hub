# Second Brain Operations

## 1. Overview

The **Second Brain** is a knowledge ingestion layer that synchronizes personal reflections and technical insights from GitHub journals into a RAG-ready PostgreSQL database. It utilizes `pgvector` for semantic search capabilities.

- **Storage**: PostgreSQL (`second_brain` table)
- **Source**: GitHub Issues (labeled `journal`)
- **Sync Tool**: `second-brain/` (Go module)
- **Journaling Format**: Uses strict singular PARA headers for categorization:
  - `## Project`: Active efforts with a deadline.
  - `## Area`: Ongoing responsibilities.
  - `## Resource`: Interests or reference material.
  - `## Archive`: Completed or inactive items.
  - `## Thought`: Fallback for general reflections (Ingested as `resource`).

---

## 2. Database Schema

The following schema defines the core knowledge table and its associated statistics view.

```sql
-- Core Knowledge Table
CREATE TYPE para_type AS ENUM (
    'project',  -- Active efforts with a deadline
    'area',     -- Ongoing responsibilities
    'resource', -- Interests or reference material
    'archive'   -- Completed or inactive items
);

CREATE TABLE second_brain (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_date DATE NOT NULL,
    content TEXT NOT NULL,
    category para_type DEFAULT 'resource',
    source_repo TEXT, -- e.g., 'observability-hub', 'cover-craft', 'personal-reading-analytics'
    origin_type TEXT,
    tags TEXT[],
    context_string TEXT,
    embedding VECTOR(1536), -- Optimized for standard OTel/OpenAI dimensions
    checksum TEXT UNIQUE,   -- Prevents duplicate ingestion
    token_count INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT check_origin_type CHECK (origin_type IN ('journal', 'adr', 'others'))
);

-- HNSW Index for high-performance vector similarity search
CREATE INDEX ON second_brain USING hnsw (embedding vector_cosine_ops);

-- Operational Stats View (Supports PARA Breakdown)
CREATE OR REPLACE VIEW second_brain_stats AS
SELECT 
    category,
    COUNT(*) as total_entries,
    MAX(entry_date) as latest_entry
FROM second_brain
GROUP BY category;
```

---

## 3. Operational Commands

### Manual Delta Sync

The sync tool automatically calculates the delta between your local DB and GitHub.

```bash
# Run the sync via the main Makefile
make brain-sync
```

### Manual Verification

To check the current status of your knowledge base:

```bash
# Check stats via kubectl (pointing specifically to the postgresql container)
kubectl exec -it statefulset/postgres-postgresql -n observability -c postgresql -- \
  psql -U server -d homelab -c "SELECT * FROM second_brain_stats;"

# Verify table structure
kubectl exec -it statefulset/postgres-postgresql -n observability -c postgresql -- \
  psql -U server -d homelab -c "\d second_brain"
```

---

## 4. Usage & Maintenance

### Sample Similarity Search (RAG)

Once embeddings are populated, you can perform semantic searches using the cosine distance operator (`<=>`).

```sql
-- Find top 5 entries most similar to a query vector
SELECT entry_date, content, 1 - (embedding <=> '[...]'::vector) AS similarity
FROM second_brain
ORDER BY embedding <=> '[...]'::vector
LIMIT 5;
```

### Notes

- **Deduplication**: Ingestion is idempotent based on the `checksum` column.
- **Vector Embeddings**: The `embedding` column is currently prepared but null. Future updates will integrate an embedding model to populate this data.
