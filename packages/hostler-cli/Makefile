# hostler-oss CLI Go build system — primary binary: hstl-oss
# Usage: make [target]
# Result: ~/.local/bin/hstl-oss

# ── variables ──
CLI_NAME    := hstl-oss
CLI_DIR     := ./cmd/hstl-oss
BIN_DIR     := ./bin
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE        := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
CLI_LDFLAGS := -s -w \
	-X 'github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss/cmd.Version=$(VERSION)' \
	-X 'github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss/cmd.Commit=$(COMMIT)' \
	-X 'github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss/cmd.Date=$(DATE)'
GOFLAGS     := -trimpath

# ── manifest paths ──
MANIFEST_DIR  := ../docs/generated
MANIFEST_FILE := $(MANIFEST_DIR)/manifest.json

# ── default targets ──
.PHONY: build test lint clean install tidy help version cross generate-manifest check-manifest check-harness-skill-drift check-heading-lint check-version-sync check-manifest-sync check-dead-links check-skill-cli-sync check-kb-refs

build: ## Build the CLI binary
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags '$(CLI_LDFLAGS)' -o $(BIN_DIR)/$(CLI_NAME) $(CLI_DIR)
	@echo "Built $(BIN_DIR)/$(CLI_NAME)"

test: ## Run all tests
	go test -count=1 ./...

lint: ## golangci-lint + staticcheck + go vet
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; fi
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; fi

clean: ## Remove build artefacts
	rm -rf $(BIN_DIR)
	go clean -cache -testcache

install: build ## Build then install hstl-oss into ~/.local/bin
	@mkdir -p $(HOME)/.local/bin
	@cp $(BIN_DIR)/$(CLI_NAME) $(HOME)/.local/bin/$(CLI_NAME).new && \
		mv $(HOME)/.local/bin/$(CLI_NAME).new $(HOME)/.local/bin/$(CLI_NAME) && \
		echo "Installed $(CLI_NAME) to $(HOME)/.local/bin/$(CLI_NAME)"

tidy: ## go mod tidy
	go mod tidy

version: build ## Print version information
	@$(BIN_DIR)/$(CLI_NAME) version

# ── manifest management ──
generate-manifest: build ## Cache --manifest output to docs/generated/manifest.json
	@mkdir -p $(MANIFEST_DIR)
	@$(BIN_DIR)/$(CLI_NAME) --manifest > $(MANIFEST_FILE)
	@echo "Generated $(MANIFEST_FILE)"

check-manifest: build ## Detect manifest.json drift (cached vs current output)
	@mkdir -p $(MANIFEST_DIR)
	@if [ ! -f $(MANIFEST_FILE) ]; then \
		echo "ERROR: $(MANIFEST_FILE) missing — run 'make generate-manifest' first"; \
		exit 1; \
	fi
	@$(BIN_DIR)/$(CLI_NAME) --manifest | diff - $(MANIFEST_FILE) > /dev/null 2>&1 || \
		(echo "DRIFT detected: manifest.json no longer matches current CLI output. Run 'make generate-manifest'"; exit 1)
	@echo "manifest.json — no drift"

# ── integration verification ──
integration-test: build check-harness-skill-drift check-heading-lint check-version-sync check-manifest-sync ## Sprint/Task lifecycle integration verification
	@bash ../scripts/integration-test.sh

check-harness-skill-drift: ## Detect drift between harness_defaults.json and Skill table
	@bash pkg/rules/scripts/check-harness-skill-drift.sh

check-heading-lint: ## Lint heading-substring anti-patterns
	@bash ../scripts/check-heading-substring-lint.sh

check-version-sync: ## Verify version sync across plugin.json / marketplace.json / CLAUDE.md
	@bash pkg/rules/scripts/check-version-sync.sh

check-manifest-sync: ## Verify hstl-oss --manifest matches Skill/Command references
	@bash pkg/rules/scripts/check-manifest-sync.sh

check-dead-links: ## Detect dead local-file references in SKILL/commands/docs
	@python3 ../scripts/check-dead-links.py

check-skill-cli-sync: ## Verify Skill/Command CLI references stay in sync with the manifest
	@python3 ../scripts/check-skill-cli-sync.py

check-kb-refs: ## Detect stale references in KB cards
	@python3 ../scripts/check-kb-references.py

# ── cross-compile ──
cross: ## Build ARM64 + AMD64 binaries
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(CLI_LDFLAGS)' -o $(BIN_DIR)/$(CLI_NAME)-linux-arm64 $(CLI_DIR)
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(CLI_LDFLAGS)' -o $(BIN_DIR)/$(CLI_NAME)-linux-amd64 $(CLI_DIR)
	@ls -lh $(BIN_DIR)/$(CLI_NAME)-linux-*

# ── help ──
help: ## Print this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
