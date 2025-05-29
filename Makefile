help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+%?:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## install luaf to the system
	@go install ./cmd/luaf

clean: uninstall
	@rm -rf ./tmp

uninstall: ## install luaf to the system
	@rm -f "$(shell which luaf)"

repl: ## run luaf repl
	@go run ./cmd/luaf

cvrg: ## show the coverage report in the browser
	go tool cover -html=./tmp/coverage.out

test: ## Run all tests
	@mkdir -p tmp
	@go test -coverprofile ./tmp/coverage.out ./...
	@go run ./cmd/luaf ./test/all.lua

bench: install ## Run limited benchmarks
	time luaf ./test/fib.lua

profile: install ## Run profiling on a fibonacci script
	@mkdir -p tmp
	@LUAF_PROFILE=./tmp/profile.pprof luaf ./test/fib.lua
	@go tool pprof -pdf ./tmp/profile.pprof > ./tmp/cpu_report.pdf
	@go tool pprof ./tmp/profile.pprof

lint: ## Run full linting rules
	@golangci-lint run
	@stylua ./test/*.lua
	@stylua ./src/runtime/lib/*.lua

update-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	cargo install stylua --features lua54
