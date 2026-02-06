# Shared Variables
NS ?= observability
KC ?= kubectl -n $(NS)
HELM ?= helm --namespace $(NS)
NIX ?= nix-shell --run

# Dynamic Nix Detection
USE_NIX = $(shell command -v nix-shell >/dev/null 2>&1 && [ -z "$$IN_NIX_SHELL" ] && echo "yes" || echo "no")

# Wrapper macro: If USE_NIX is yes, re-run make with the same goals inside nix-shell and exit.
ifeq ($(USE_NIX),yes)
NIX_WRAP = @$(NIX) "make $(MAKECMDGOALS)" && exit 0
else
NIX_WRAP =
endif

# Tooling
LINT_IMAGE = ghcr.io/igorshubovych/markdownlint-cli:v0.44.0

.PHONY: adr lint

# Architecture Decision Record Creation
adr:
	@./scripts/create_adr.sh

# Markdown Linting
lint:
	docker run --rm -v "$(PWD):/data" -w /data $(LINT_IMAGE) --fix "**/*.md"