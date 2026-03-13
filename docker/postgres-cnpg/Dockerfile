FROM ghcr.io/cloudnative-pg/postgresql:17

USER root

# Install the Golden Trio of extensions (aligned with existing successful build)
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-17-postgis-3 \
    postgresql-17-postgis-3-scripts \
    postgresql-17-pgvector \
    postgresql-17-timescaledb \
    && rm -rf /var/lib/apt/lists/*

# Return to the standard postgres user
USER 26
