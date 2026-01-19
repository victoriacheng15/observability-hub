# Proxy Service Architecture

The Proxy Service (`proxy/`) is a custom Go application that acts as the API gateway and Data Pipeline engine for the platform. It bridges external data sources with the PostgreSQL (TimescaleDB) storage.

## Component Details

### API Overview

| Endpoint | Method | Purpose |
| :--- | :--- | :--- |
| `/` | GET | Returns a JSON welcome message. |
| `/api/reading` | GET | Placeholder for future reading retrieval features. |
| `/api/sync/reading` | GET | Synchronizes reading data from MongoDB to PostgreSQL (TimescaleDB). |

### Endpoint Details

#### Data Pipeline Engine (`/api/sync/reading`)

This endpoint triggers the extraction, transformation, and loading of data.

1. **Connect**: Establishes connection to MongoDB using `MONGO_URI`.
2. **Query**: Finds documents in the source collection where `status="ingested"`.
3. **Transform**: Converts documents into a standardized JSONB format.
4. **Load**: Inserts records into the PostgreSQL (TimescaleDB) `reading_analytics` table.
5. **Update**: Marks the original MongoDB documents as `status="processed"`.

## Data Flow: Analytical Data Pipeline

```mermaid
sequenceDiagram
    participant Mongo as MongoDB
    participant Proxy as Proxy Service
    participant PG as PostgreSQL

    Proxy->>Mongo: Find docs (status="ingested")
    Mongo-->>Proxy: Return batch
    Proxy->>Proxy: Transform to JSONB
    Proxy->>PG: INSERT into reading_analytics
    Proxy->>Mongo: Update status="processed"
```
