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

cvrg: ## show the coverage report in the browser
	go tool cover -html=./tmp/coverage.out

test: lint ## Run all tests
	@mkdir -p tmp
	@go test -coverprofile ./tmp/coverage.out ./...
	@go run ./cmd/luaf ./test/all.lua

bench: install ## Run limited benchmarks and profiling
	@mkdir -p tmp
	@echo "=== non-tailcall ==="
	@LUAF_PROFILE=./tmp/profile.pprof time luaf ./test/profile/fib.lua
	@echo "=== tailcall ==="
	@time luaf ./test/profile/fibt.lua
	@go tool pprof -pdf ./tmp/profile.pprof > ./tmp/cpu_report.pdf

lint: ## Run full linting rules
	@golangci-lint run
	@stylua ./test/*.lua
	@stylua ./src/runtime/lib/*.lua

update-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	cargo install stylua --features lua54

docs: ## Run the docs site
	@cd docs && bundle exec jekyll serve --drafts
