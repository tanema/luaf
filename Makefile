help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+%?:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## install luaf to the system
	@go install ./cmd/luaf

clean: uninstall
uninstall: ## install luaf to the system
	@rm -f "$(shell which luaf)"

repl: ## run luaf repl
	@go run ./cmd/luaf

cvrg: ## show the coverage report in the browser
	go tool cover -html=./tmp/coverage.out

test: test-go test-lua lint ## Run all tests
test-go: # Run only go tests
	@mkdir -p tmp
	@go test -coverprofile ./tmp/coverage.out ./...

test-lua: # Run tests interpreting lua
	@go run ./cmd/luaf ./test/all.lua

bench: install ## Run limited benchmarks
	time luaf ./test/fib.lua

dbg: ## Run build version of luaf on the scratch script
	@./tools/luaf

profile: install ## Run profiling on a fibonacci script
	@mkdir -p tmp
	@LUAF_PROFILE=./tmp/profile.pprof luaf ./test/fib.lua
	@go tool pprof -pdf ./tmp/profile.pprof > ./tmp/cpu_report.pdf
	@go tool pprof ./tmp/profile.pprof

### Linting
lint: lint-vet lint-ci lint-staticcheck lint-lua## Run full linting rules
lint-vet:
	@go vet ./...
lint-ci:
	@golangci-lint run
lint-staticcheck:
	@staticcheck ./...
lint-lua:
	@stylua ./test/*.lua
	@stylua ./lib/*.lua
