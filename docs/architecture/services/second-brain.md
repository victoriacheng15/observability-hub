# Second Brain Service Architecture

The Second Brain Service (`services/second-brain/`) is a knowledge ingestion layer that transforms GitHub-based journaling entries into atomic, searchable database records.

## 🧩 Component Details

- **Type**: Go sync utility.
- **Source**: GitHub Issues (labeled `journal`).
- **Destination**: PostgreSQL (`second_brain` table) with **pgvector** support.
- **Philosophy**: Atomic knowledge decomposition via the **PARA** (Project, Area, Resource, Archive) method.

## ⚙️ Logic Flow

1. **Check State**: Queries the local database via **PostgresWrapper** to find the `entry_date` of the latest ingested thought.
2. **Fetch Delta**: Uses the GitHub CLI (`gh`) to retrieve new issues created after the latest date.
3. **Atomize**: Parses Markdown PARA headers and decomposes the issue body into individual `AtomicThought` objects.
4. **Enrich**: Automatically generates semantic tags and calculates token counts for future LLM/RAG use.
5. **Persist**: Saves atoms into PostgreSQL using standardized infrastructure spans.

## 🔭 Observability Implementation

- **Root Span**: `job.second_brain_sync` identifies the sync operation.
- **Grouping Spans**: `ingest.delta` wraps the processing of an individual GitHub issue.
- **Performance Spans**: `parse.markdown.duration` monitors the complexity of knowledge decomposition.
- **Usage Metrics**: `second.brain.token.count.total` tracks the data volume for cost/capacity planning.
- **Infrastructure Visibility**: Inherits `db.pool.wait_time` and `db.postgres.*` spans from the wrapper library.
