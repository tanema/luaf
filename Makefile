.PHONY: test docs
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+%?:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: test lint ## run all testing and linting

install: ## install luaf to the system
	@go install ./cmd/luaf

clean: uninstall
	@rm -rf ./tmp

uninstall: ## uninstall luaf from the system
	@rm -f "$(shell which luaf)"

repl: ## run luaf repl
	@go run ./cmd/luaf

test: clean test/go test/lua ## Run all tests

test/go:
	@mkdir -p ./tmp/cover
	@go test -coverprofile ./tmp/cover/unit.out ./...

test/lua:
	@mkdir -p ./tmp/cover
	@go build -cover -o ./tmp/luaf ./cmd/luaf
	@GOCOVERDIR=./tmp/cover ./tmp/luaf ./test/all.lua
	@go tool covdata percent -i=./tmp/cover

bench: install ## Run limited benchmarks and profiling
	@mkdir -p tmp
	@echo "=== non-tailcall ==="
	@LUAF_PROFILE=./tmp/profile.pprof time luaf ./test/profile/fib.lua
	@echo "=== tailcall ==="
	@time luaf ./test/profile/fibt.lua
	@go tool pprof -pdf ./tmp/profile.pprof > ./tmp/cpu_report.pdf

lint: lint/go lint/lua ## Run full linting rules

lint/go:
	@golangci-lint run

lint/lua:
	@stylua ./test/*.lua
	@stylua ./src/runtime/lib/*.lua

docs: ## Run the docs site
	@cd docs && bundle exec jekyll serve --drafts
