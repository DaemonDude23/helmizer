# Task entry points. Every target runs inside the flake devShell (the one
# toolchain pin, flake.lock), whether or not you are already in it.
.DEFAULT_GOAL := help
SHELL := bash
.SHELLFLAGS := -euo pipefail -c
NIX ?= nix
# Already in `nix develop` / direnv? Then run tools directly.
RUN := $(if $(IN_NIX_SHELL),,$(NIX) develop --command)
UNIT ?= github

.PHONY: help
help: ## List targets
	@grep -hE '^[a-zA-Z0-9_.-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: check
check: lint flake-check ## Everything CI's validate stage runs (lint + nix flake check)

.PHONY: lint
lint: ## Run every pre-commit hook on every file
	$(RUN) pre-commit run --all-files --show-diff-on-failure

.PHONY: fmt
fmt: ## Format the whole tree (treefmt)
	$(NIX) fmt

.PHONY: flake-check
flake-check: ## Build every package, test, and formatting check
	$(NIX) flake check --print-build-logs

.PHONY: renovate-check
renovate-check: ## Validate renovate.json strictly
	$(NIX) shell --inputs-from . nixpkgs#renovate --command renovate-config-validator --strict renovate.json

.PHONY: secrets
secrets: ## Scan the full git history for secrets
	$(RUN) gitleaks git --redact --no-banner .

.PHONY: update
update: ## Update flake.lock (every tool at once)
	$(NIX) flake update

.PHONY: tofu-plan
tofu-plan: ## terragrunt plan for infra/tofu/$(UNIT) (read-only)
	cd infra/tofu/$(UNIT) && $(TOFU_ENV) $(RUN) terragrunt plan

.PHONY: tofu-apply
tofu-apply: ## terragrunt apply of the plan written by tofu-plan
	cd infra/tofu/$(UNIT) && $(TOFU_ENV) $(RUN) terragrunt apply

.PHONY: tofu-validate
tofu-validate: ## tofu validate for every infra/tofu unit (downloads providers)
	@for d in infra/tofu/*/; do \
		echo "== $$d"; \
		(cd "$$d" && $(RUN) tofu init -backend=false -input=false >/dev/null && $(RUN) tofu validate) || exit 1; \
	done

.PHONY: release
release: ## Tag v$(VERSION) (signed) after checking src/VERSION and CHANGELOG.md; push the tag to publish
	@v=$$(cat src/VERSION); \
	test -z "$$(git status --porcelain)" || { echo "working tree not clean" >&2; exit 1; }; \
	grep -q "^## \[$$v\]" CHANGELOG.md || { echo "CHANGELOG.md has no [$$v] section" >&2; exit 1; }; \
	git tag -s "v$$v" -m "v$$v" && echo "tagged v$$v; publish with: git push origin v$$v"

# The github provider reads GITHUB_TOKEN; take it from the gh CLI login.
TOFU_ENV = GITHUB_TOKEN=$$(gh auth token)

.PHONY: docs
docs: ## Serve the docs with live reload (http://127.0.0.1:8000)
	$(NIX) run .#docs

.PHONY: docs-build
docs-build: ## Strict static docs build into ./result
	$(NIX) build .#docs --print-build-logs

GO_DIR := src

.PHONY: build
build: ## Build the binary with Nix (./result/bin/helmizer)
	$(NIX) build --print-build-logs

.PHONY: test
test: ## go test with the race detector
	cd $(GO_DIR) && $(RUN) go test -race -count=1 ./...

.PHONY: cover
cover: ## Test coverage report (coverage.html)
	cd $(GO_DIR) && $(RUN) go test -coverprofile=coverage.out ./... && $(RUN) go tool cover -html=coverage.out -o coverage.html

.PHONY: fuzz
fuzz: ## Run every fuzz target for FUZZTIME (default 15s)
	cd $(GO_DIR) && for p in $$($(RUN) go list ./...); do \
		for t in $$($(RUN) go test -list '^Fuzz' "$$p" | grep '^Fuzz' || true); do \
			$(RUN) go test -run='^$$' -fuzz="^$$t\$$" -fuzztime=$${FUZZTIME:-15s} "$$p" || exit 1; \
		done; done

.PHONY: vuln
vuln: ## govulncheck + gosec
	cd $(GO_DIR) && $(RUN) govulncheck ./... && $(RUN) gosec -quiet -severity medium -confidence medium ./...

.PHONY: vendor-hash
vendor-hash: ## Refresh flake.nix vendorHash after go.mod/go.sum changes
	$(RUN) scripts/update-vendor-hash.sh

.PHONY: snapshot
snapshot: ## Local goreleaser snapshot build into ./dist
	$(RUN) goreleaser release --snapshot --clean --skip=sign,sbom
