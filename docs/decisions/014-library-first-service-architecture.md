# ADR 014: Library-First Service Architecture

- **Status:** Accepted
- **Date:** 2026-02-16
- **Author:** Victoria Cheng

## Context and Problem Statement

As the Observability Hub matures, it is transitioning from a collection of isolated services into a multi-interface platform. We now face the need to support multiple execution triggers for the same business logic, such as automated background ingestion (systemd/k3s) and real-time requests via the `proxy`.

The current architecture tightly couples business logic (e.g., metric collection, "Second Brain" ingestion) within the `main` packages of individual services. This makes it difficult to reuse logic across different interfaces without duplication or fragile dependencies.

## Decision Outcome

Adopt a **Library-First Service Architecture**. This involves extracting all core domain logic into a highly modular `pkg/` directory and reorganizing the root structure to separate reusable libraries from executable services.

### The Strategy

- **Module Extraction**: Relocate core logic to dedicated packages in `pkg/` (e.g., `pkg/metrics`, `pkg/brain`).
- **Service Relocation**: Move all standalone binaries into a `services/` directory (e.g., `services/proxy`, `services/system-metrics`).
- **Interface Decoupling**: Ensure `pkg/` libraries are transport-agnostic (no HTTP or CLI logic), allowing them to be imported by any service or tool within the hub.
- **Consistent Initialization**: Standardize how shared resources (OTel, Database connections) are initialized across the platform.

### Rationale

- **Consistency**: Guarantees that different entry points (CLI, API, Background Tasks) execute the exact same business logic.
- **Maintainability**: Reduces root directory noise and enforces a clear "Paved Road" for adding new capabilities.
- **Testability**: Facilitates 80%+ unit test coverage by isolating business logic from external side effects and transport layers.
- **Scalability**: Allows the platform to support new "access patterns" (like a TUI or Mobile API) simply by importing the relevant `pkg/` modules.

## Consequences

### Positive

- **Unified Logic**: One "Source of Truth" for all platform operations.
- **Improved DX**: Clearer project structure makes it easier for developers to navigate and contribute to the codebase.
- **Robust Automation**: Background tasks and interactive tools benefit from the same hardened logic.

### Negative

- **Initial Refactoring Effort**: Requires significant movement of files and updates to import paths, Makefiles, and CI/CD pipelines.
- **Increased Boilerplate**: Each new feature requires a separate library in `pkg/` and a corresponding registration in `services/`.

## Verification

- [x] **Structural Reorg**: `services/` and `pkg/` directories populated and following the new convention.
- [x] **Library Extraction**: Core logic from `second-brain` and `system-metrics` successfully ported to `pkg/`.
- [x] **Build Integrity**: All services build and test successfully from their new locations.
