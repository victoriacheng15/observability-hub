# ADR 004: Spatial Keyboard Telemetry Pipeline

- **Status:** Superseded
- **Date:** 2026-01-03
- **Author:** Victoria Cheng

## Context and Problem Statement

Standard input monitoring (keyloggers or counters) lacks the physical context of *where* interactions happen. For ergonomic analysis, heatmapping, and advanced hardware telemetry, we need a way to map raw Linux input events to physical coordinates on a 2D plane.

Existing solutions are either platform-specific (Windows-only), high-latency (Python-based), or closed-source. We need a low-level, high-performance bridge that can ship this data from an "Edge" device (Desktop) to a "Control Plane" (Laptop) without impacting system performance.

## Decision Outcome

A distributed IoT pipeline consisting of three tiers:

- **C++ Edge Agent:** Reads `/dev/input/event*` directly using `ioctl`. Maps scancodes to `(x, y)` coordinates in millimeters.
- **Go Proxy (Gateway):** A centralized ingress point that validates telemetry payloads and batches them for database insertion.
- **PostGIS (Storage):** A relational database with spatial indexing to store keypresses as `GEOMETRY(POINT)`.

### Rationale

- **C++ for Performance:** Direct kernel access and zero Garbage Collection (GC) ensures that even high-speed typing (150+ WPM) doesn't cause telemetry lag or CPU spikes.
- **Distributed Architecture:** Decouples the data collection (Desktop) from the visualization (Laptop), allowing the observability hub to remain centralized.
- **Spatial Focus:** By using PostGIS, we can perform advanced spatial queries (e.g., "distance traveled between keypresses") that are impossible in standard time-series databases.

## Consequences

### Positive

- **Performance:** Near-zero CPU / < 5MB RAM usage on the edge.
- **Latency:** Deterministic behavior (no GC pauses).
- **Analysis:** Enables spatial queries via PostGIS.

### Negative

- **Complexity:** Requires managing a C++ build chain alongside Go.
- **Portability:** Tightly coupled to Linux `ioctl` and specific hardware layouts.

## Planned Verification

- [ ] **Manual Check:** Verify keypresses appear on the spatial heatmap.
- [ ] **Automated Tests:** Unit tests for scancode-to-coordinate mapping.
