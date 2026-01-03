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

## 📝 RFC Template

To create a new proposal, copy the block below into a new `.md` file.

```markdown
# RFC [00X]: [Descriptive Title]

**Status:** Proposed | Accepted | Superseded  
**Date:** YYYY-MM-DD  
**Author:** Victoria Cheng

## The Problem

Identify the issue or opportunity. Why does this need to change?

## Proposed Solution

Technical details of the implementation. Use code snippets or diagrams.

## Comparison / Alternatives Considered

What else could we have done? Why is this path better?

## Failure Modes (Operational Excellence)

How does this break? How will we know when it's failing in production?

## Conclusion

Final summary and next steps.
```
