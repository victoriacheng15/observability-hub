# Shared Variables
NS ?= observability
KC ?= kubectl -n $(NS)
HELM ?= helm --namespace $(NS)
NIX ?= nix-shell --run

# Tooling
LINT_IMAGE = ghcr.io/igorshubovych/markdownlint-cli:v0.44.0

.PHONY: adr nix-% lint

# Architecture Decision Record Creation
adr:
	@./scripts/create_adr.sh

# Markdown Linting
lint:
	docker run --rm -v "$(PWD):/data" -w /data $(LINT_IMAGE) --fix "**/*.md"

# Run any target inside nix-shell (Generic Wrapper)
nix-%:
	@$(NIX) "make $*"