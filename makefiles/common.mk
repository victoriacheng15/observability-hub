# Shared Variables
NS ?= observability
KC ?= kubectl -n $(NS)
HELM ?= helm --namespace $(NS)

# Dynamic Nix Detection
USE_NIX = $(shell if command -v nix-shell >/dev/null 2>&1 && [ -z "$$IN_NIX_SHELL" ] && [ "$$GITHUB_ACTIONS" != "true" ]; then echo "yes"; else echo "no"; fi)

ifeq ($(USE_NIX),yes)
    NIX_RUN = nix-shell --run
    # NIX_WRAP: If Nix is available and we aren't in a shell, re-run the entire target inside nix-shell
    # The 'exit $$?' ensures the outer make stops immediately with the inner make's exit code.
    NIX_WRAP = @$(NIX_RUN) "make $(MAKECMDGOALS)" && exit $$?
else
    NIX_RUN = bash -c
    NIX_WRAP = @
endif

# Tooling
LINT_IMAGE = ghcr.io/igorshubovych/markdownlint-cli:v0.44.0

.PHONY: adr lint lint-configs

# Architecture Decision Record Creation
adr:
	@./scripts/create_adr.sh

# Markdown Linting
lint:
	docker run --rm -v "$(PWD):/data" -w /data $(LINT_IMAGE) --fix "**/*.md"

# Configuration Linting (HCL & GitHub Actions)
lint-configs:
	@echo "Formatting OpenBao policies..."
	$(NIX_RUN) "bao policy fmt policies/app-policy.hcl"
	@echo "Validating GitHub Actions workflows..."
	$(NIX_RUN) "action-validator .github/workflows/*.yml"