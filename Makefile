# ── Co-Learning Agent ────────────────────────────────────────────────────────
# Go 1.24 project. Source lives under src/; templates are embedded by path
# (src/templates), so run/serve targets execute from the repo root.
#
# Cheatsheet below in `make help`.

GO      ?= go
PORT    ?= 8080
# Durable learning snapshot location (Phase 2). `off` disables persistence.
# Override with: make run LEARNING_STORE_PATH=data/learning-state.json
LEARNING_STORE_PATH ?= data/learning-state.json

BINARY_DIR := bin
BINARY     := $(BINARY_DIR)/colearning-api

# Project packages (avoid sweeping up test fixtures / vendored dirs)
PKGS := ./src/...

.PHONY: help
help: ## Show this cheatsheet
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \x1b[36m<target>\x1b[0m\n\nTargets:\n"} \
		/^[a-zA-Z0-9_.-]+:.*##/ {printf "  \x1b[36m%-22s\x1b[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\nRun-time overrides:\n  make run PORT=9000 LEARNING_STORE_PATH=off\n\n"

# ── Build ────────────────────────────────────────────────────────────────────
.PHONY: build
build: ## Build the API binary into bin/
	$(GO) build -o $(BINARY) ./src/cmd/api

.PHONY: run
run: ## Start the dev server (Ctrl+C for graceful shutdown + snapshot flush)
	LEARNING_STORE_PATH=$(LEARNING_STORE_PATH) $(GO) run ./src/cmd/api

# ── Quality ─────────────────────────────────────────────────────────────────
.PHONY: check
check: fmt-check vet test ## fmt-check + vet + test (pre-commit / CI gate)

.PHONY: test
test: ## Run all tests
	$(GO) test $(PKGS)

.PHONY: test-cover
test-cover: ## Run tests with coverage summary
	$(GO) test -cover $(PKGS)

.PHONY: vet
vet: ## go vet
	$(GO) vet $(PKGS)

.PHONY: fmt
fmt: ## Format all source files in place
	$(GO) fmt $(PKGS)

.PHONY: fmt-check
fmt-check: ## Fail if any file is unformatted
	@out=$$(gofmt -l src/); \
	if [ -n "$$out" ]; then printf "unformatted files:\n%s\n" "$$out"; exit 1; fi

# ── Data (durable learning snapshot) ────────────────────────────────────────
.PHONY: data-clean
data-clean: ## Delete the durable learning snapshot (next boot reseeds)
	rm -rf data

# ── Smoke test ──────────────────────────────────────────────────────────────
.PHONY: smoke
smoke: ## Boot server, post a tutor turn, restart, verify restore
	@./scripts/smoke-persistence.sh

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BINARY_DIR)
