.PHONY: bootstrap build test smoke install-hooks setup agent-install run run-local run-binary clean

GO_BIN := $(shell bash scripts/bootstrap-go.sh)

bootstrap:
	bash scripts/bootstrap-go.sh

build:
	mkdir -p .codex-mcp/bin
	$(GO_BIN) build -o .codex-mcp/bin/codex-mcp ./cmd/codex-mcp

test:
	$(GO_BIN) test ./...

smoke: build
	bash scripts/smoke-workflow.sh

install-hooks:
	bash scripts/install-hooks.sh

setup: build test smoke install-hooks

agent-install:
	bash scripts/install-agent.sh

run: build
	$(GO_BIN) run ./cmd/codex-mcp

run-local: build
	bash scripts/run-local.sh

run-binary: build
	.codex-mcp/bin/codex-mcp

clean:
	rm -rf .codex-mcp
