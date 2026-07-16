.PHONY: test docs
SHELL=/bin/zsh -o pipefail

help: ## Show this help.
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

test: clean test/go test/lua ## Run all tests
	@echo "══📊 \033[36mCoverage Report\033[0m══════════════════════════════════════════════════"
	@go tool covdata percent -i=tmp/coverage/unit,tmp/coverage/integration -o=tmp/coverage/all.out
	@go tool cover -html=tmp/coverage/all.out -o tmp/coverage/index.html
	@echo "coverage report at: file://${PWD}/tmp/coverage/index.html"

test/go:
	@echo "══🦫 \033[36mGo Tests\033[0m════════════════════════════════════════════════════════"
	@mkdir -p ./tmp/coverage/unit
	@go test -cover ./... -args -test.gocoverdir="${PWD}/tmp/coverage/unit"

test/lua:
	@echo "══⚙️ \033[36mLua Tests\033[0m═══════════════════════════════════════════════════════"
	@mkdir -p ./tmp/coverage/integration
	@go build -cover -o ./tmp/luaf ./cmd/luaf
	@GOCOVERDIR=./tmp/coverage/integration ./tmp/luaf ./test/all.lua

bench: install ## Run limited benchmarks and profiling
	@mkdir -p tmp
	@echo "══ non-tailcall ═════════════════════════════════════════════════════════════════════"
	@time luaf ./test/profile/fib.lua
	@echo "══ tailcall ═════════════════════════════════════════════════════════════════════════"
	@time luaf ./test/profile/fibt.lua

lint: lint/go lint/lua ## Run full linting rules

lint/go:
	@echo "══🔎 \033[36mLint Go\033[0m══════════════════════════════════════════════════════════"
	@golangci-lint run

lint/lua:
	@echo "══🔎 \033[36mLint Lua\033[0m═════════════════════════════════════════════════════════"
	@stylua --check \
		--syntax=Lua54 \
		--output-format=summary ./**/*.lua 

docs: ## Run the docs site
	@cd docs && \
		bundle install && \
		bundle exec jekyll serve --drafts

scratch: ## Run my scratch file where I do my lil tests
	@go run ./cmd/luaf -l ./test/misc/scratch.lua

compare: ## Compare bytecode output
	@echo "══LUAF═════════════════════════════════════════════════════════════════════════"
	@go run ./cmd/luaf -l -p ./test/misc/scratch.lua
	@echo "══LUA══════════════════════════════════════════════════════════════════════════"
	@luac -l ./test/misc/scratch.lua
	@rm luac.out
