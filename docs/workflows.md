# GitHub Actions Workflows

This document details the CI/CD and automation pipelines configured in `.github/workflows/`. These workflows provide the "Golden Path" for testing, linting, and delivering the Observability Hub.

---

## 📂 Core Workflows

### 🚢 [GitHub Pages Deployment](../.github/workflows/deploy.yml)

Handles the automated build and hosting of the public-facing portfolio page.

- **Trigger**: Pushes to the main branch affecting the page generator directory or manual trigger.
- **Responsibility**: Sets up the Go environment, builds the site generator, executes it to generate static assets, and deploys the output to the public environment.
- **Key Feature**: Leverages native GitHub Actions for seamless artifact management and hosting.

### 🧪 [Go Lint & Test](../.github/workflows/go-ci.yml)

Ensures code quality and functional correctness across all Go modules in the monorepo.

- **Trigger**: Any Push or Pull Request affecting Go source code.
- **Responsibility**: Verifies code formatting, runs static analysis, and executes the full suite of unit and integration tests.
- **Key Feature**: Centralized cache management for all module dependencies to ensure fast and consistent CI runs.

### 🏗️ [Infrastructure Linting](../.github/workflows/infra-lint.yml)

Validates Kubernetes manifests to ensure security and operational best practices.

- **Trigger**: Pushes or Pull Requests affecting the `k3s/` directory.
- **Responsibility**: Scans K3s manifests using `kube-linter` to catch configuration errors early.
- **Key Feature**: Automated enforcement of infrastructure-as-code quality standards.

### 🤝 [Label-Based PR Merge](../.github/workflows/label-merge.yml)

Facilitates "Fleet Commander" style delivery through automated governance.

- **Trigger**: Labeling a Pull Request with the designated merge label.
- **Responsibility**: Automatically squashes and merges the PR into the main branch.
- **Key Feature**: Enables automated delivery to the device fleet by transitioning from "Review" to "Deployed" via a single UI interaction.

### 📝 [Markdown Linter](../.github/workflows/markdownlint.yml)

Enforces documentation standards and protects the "Institutional Memory."

- **Trigger**: Pull Requests affecting Markdown files or manual trigger.
- **Responsibility**: Scans the repository for documentation styling violations to ensure consistency.
- **Key Feature**: Enforces standards across all ADRs, RFCs, and operational guides.

### 🛡️ [Security Scan](../.github/workflows/security.yml)

Proactively identifies vulnerabilities in the Go codebase and dependencies.

- **Trigger**: Pushes or Pull Requests affecting Go code, and weekly scheduled runs.
- **Responsibility**: Executes `govulncheck` to scan for known security vulnerabilities.
- **Key Feature**: Multi-layered protection through both event-driven and periodic security auditing.

---

## 🛠️ Reusable Toolkit

To ensure consistency and reduce boilerplate, standardized workflows (Merging, Linting) are powered by the **[platform-actions](https://github.com/victoriacheng15hub/platform-actions)** toolkit. This centralizes governance and allows for rapid updates to delivery standards across the entire ecosystem.
