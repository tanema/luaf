.PHONY: test docs

help: ## Show this help.
	@echo "╔════════════════════════════════════════╗"
	@echo "║            🤠 \033[36mLuaf\033[0m                     ║"
	@echo "╠════════════════════════════════════════╣"
	@echo "║ \033[1;97mUsage:\033[0m \033[36mmake\033[0m target                     ║"
	@echo "╠════════════════════════════════════════╝"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+%?:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "║ \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo "╚═════════════════════════════════════════"

install: ## install luaf to the system
	@go install ./cmd/luaf

clean: uninstall
	@rm -rf ./tmp

uninstall: ## uninstall luaf from the system
	@rm -f "$(shell which luaf)"

repl: ## run luaf repl
	@go run ./cmd/luaf

test: clean test/go test/lua lint/go lint/lua ## Run all tests
	@echo "╠════════════════════════════════════════╗"
	@echo "║           📊 \033[36mCoverage Report\033[0m           ║"
	@echo "╠════════════════════════════════════════╝"
	@go tool covdata percent \
		-i=./tmp/coverage/unit,./tmp/coverage/integration \
		| sed 's/^/║ /'
	@echo "╚════════════════════════════════════════¤"

test/go:
	@echo "╔════════════════════════════════════════╗"
	@echo "║              🦫 \033[36mGo Tests\033[0m               ║"
	@echo "╠════════════════════════════════════════╝"
	@mkdir -p ./tmp/coverage/unit
	@go test -cover ./... -args \
		-test.gocoverdir="${PWD}/tmp/coverage/unit" \
		| sed 's/^/║ /'

test/lua:
	@echo "╠════════════════════════════════════════╗"
	@echo "║              ⚙️ \033[36mLua Tests\033[0m              ║"
	@echo "╠════════════════════════════════════════╝"
	@mkdir -p ./tmp/coverage/integration
	@go build -cover -o ./tmp/luaf ./cmd/luaf
	@GOCOVERDIR=./tmp/coverage/integration ./tmp/luaf ./test/all.lua | sed 's/^/║ /'

bench: install ## Run limited benchmarks and profiling
	@mkdir -p tmp
	@echo "╔═ non-tailcall ═════════════════════════"
	@time luaf ./test/profile/fib.lua
	@echo "╠═ tailcall ═════════════════════════════"
	@time luaf ./test/profile/fibt.lua
	@echo "╚════════════════════════════════════════¤"

lint: lint/go lint/lua ## Run full linting rules
	@echo "╚════════════════════════════════════════¤"

lint/go:
	@echo "╠════════════════════════════════════════╗"
	@echo "║            🔎 \033[36mLint Go\033[0m                  ║"
	@echo "╠════════════════════════════════════════╝"
	@golangci-lint run | sed 's/^/║ /'

lint/lua:
	@echo "╠════════════════════════════════════════╗"
	@echo "║            🔎 \033[36mLint Lua\033[0m                 ║"
	@echo "╠════════════════════════════════════════╝"
	@stylua --check \
		--output-format=summary ./test/*.lua ./src/runtime/lib/*.lua ./src/runtime/lib/*.lua \
		| sed 's/^/║ /'

docs: ## Run the docs site
	@cd docs && bundle exec jekyll serve --drafts
