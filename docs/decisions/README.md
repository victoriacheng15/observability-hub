# Architectural Decision Records (ADR)

This directory serves as the **Institutional Memory** for the Observability Hub. It documents the "Why" behind major technical choices, ensuring the project remains maintainable and its evolution is transparent.

---

## Decision Lifecycle

| Status | Meaning |
| :--- | :--- |
| **🟢 Proposed** | Planning phase. The design is being discussed or researched. |
| **🔵 Accepted** | Implementation phase or completed. This is the current project standard. |
| **🟡 Superseded** | Historical record. This decision has been replaced by a newer ADR. |

---

## Conventions

- **File Naming:** `00X-descriptive-title.md`
- **Dates:** Use ISO 8601 format (`YYYY-MM-DD`).
- **Formatting:** Use hyphens (`-`) for all lists; no numbered lists.
- **Automation:** Run `make rfc` to interactively generate a new file that follows these standards.

---

## 📝 ADR Template

To create a new proposal, copy the block below into a new `.md` file.

```markdown
# ADR [00X]: [Descriptive Title]

- **Status:** Proposed | Accepted | Superseded
- **Date:** YYYY-MM-DD
- **Author:** Victoria Cheng

## Context and Problem Statement

What specific issue triggered this change?

## Decision Outcome

What was the chosen architectural path?

## Consequences

- **Positive:** (e.g., Faster development, resolved dependency drift).
- **Negative/Trade-offs:** (e.g., Added complexity to the CI/CD pipeline).

## Verification

- [ ] **Manual Check:** (e.g., Verified logs/UI locally).
- [ ] **Automated Tests:** (e.g., `make nix-go-test` passed).

```
