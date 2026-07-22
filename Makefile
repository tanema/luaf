.PHONY: test docs
SHELL=/bin/zsh -o pipefail

help:
	@echo "╔═══════════════════════════════════════════════════════════════════════════════════╗"
	@echo "║ 🤠 \033[36mLuaf\033[0m lua for fun and laufs                                                     ║"
	@echo "╠═══════════════════════════════════════════════════════════════════════════════════╣"
	@echo "║ \033[1;97mUsage:\033[0m \033[36mmake\033[0m target                                                                ║"
	@echo "╠═══════════════════════════════════════════════════════════════════════════════════╣"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+%?:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "║ \033[36m%-20s\033[0m %-60s ║\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo "╚═══════════════════════════════════════════════════════════════════════════════════╝"

install: ## install luaf to the system
	@go install ./cmd/luaf

clean: uninstall ## uninstall and clean artifacts
	@rm -rf ./tmp

uninstall: ## uninstall luaf from the system
	@rm -f "$(shell which luaf)"

repl: ## run luaf repl
	@go run ./cmd/luaf

all: test lint ## Run tests and linting

test: ## Run all tests
	@mkdir -p ./tmp/coverage/unit
	@echo "══ Go Test ══════════════════════════════════════════════════════════════════════════"
	@go test -cover ./... -args -test.gocoverdir="${PWD}/tmp/coverage/unit"
	@mkdir -p ./tmp/coverage/integration
	@go build -cover -o ./tmp/luaf ./cmd/luaf
	@echo "══ Lua Test ═════════════════════════════════════════════════════════════════════════"
	@GOCOVERDIR=./tmp/coverage/integration ./tmp/luaf ./test/all.lua
	@go tool covdata percent -i=tmp/coverage/unit,tmp/coverage/integration -o=tmp/coverage/all.out
	@go tool cover -html=tmp/coverage/all.out -o tmp/coverage/index.html
	@echo "coverage report at: file://${PWD}/tmp/coverage/index.html"

bench: install ## Run limited benchmarks and profiling
	@mkdir -p tmp
	@echo "══ non-tailcall ═════════════════════════════════════════════════════════════════════"
	@time luaf ./test/profile/fib.lua
	@echo "══ tailcall ═════════════════════════════════════════════════════════════════════════"
	@time luaf ./test/profile/fibt.lua

lint: ## Run all linting tooling
	@golangci-lint run
	@stylua --check --syntax=Lua54 --output-format=summary .

docs: ## Run the docs site
	@cd docs && \
		bundle install && \
		bundle exec jekyll serve --drafts

scratch:
	@go run ./cmd/luaf -l ./test/misc/scratch.lua
