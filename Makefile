.PHONY: test docs
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+%?:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## install luaf to the system
	@go install ./cmd/luaf

clean: uninstall
	@rm -rf ./tmp

uninstall: ## uninstall luaf from the system
	@rm -f "$(shell which luaf)"

repl: ## run luaf repl
	@go run ./cmd/luaf

test: clean test/go test/lua lint/go lint/lua ## Run all tests
	@echo "📊 Coverage Report"
	@echo "=================="
	@go tool covdata percent -i=./tmp/coverage/unit,./tmp/coverage/integration

test/go:
	@echo "🦫 Go Tests"
	@echo "==========="
	@mkdir -p ./tmp/coverage/unit
	@go test -cover ./... -args -test.gocoverdir="${PWD}/tmp/coverage/unit"
	@echo "----------"

test/lua:
	@echo "⚙️ Lua Tests"
	@echo "============"
	@mkdir -p ./tmp/coverage/integration
	@go build -cover -o ./tmp/luaf ./cmd/luaf
	@GOCOVERDIR=./tmp/coverage/integration ./tmp/luaf ./test/all.lua

bench: install ## Run limited benchmarks and profiling
	@mkdir -p tmp
	@echo "=== non-tailcall ==="
	@LUAF_PROFILE=./tmp/profile.pprof time luaf ./test/profile/fib.lua
	@echo "=== tailcall ==="
	@time luaf ./test/profile/fibt.lua
	@go tool pprof -pdf ./tmp/profile.pprof > ./tmp/cpu_report.pdf

lint: lint/go lint/lua ## Run full linting rules

lint/go:
	@echo "🔎 Lint Go"
	@echo "=========="
	@golangci-lint run
	@echo "----------"

lint/lua:
	@echo "🔎 Lint Lua"
	@echo "==========="
	@stylua --check --output-format=summary ./test/*.lua ./src/runtime/lib/*.lua ./src/runtime/lib/*.lua
	@echo "-----------"

docs: ## Run the docs site
	@cd docs && bundle exec jekyll serve --drafts
