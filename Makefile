help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+%?:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: ## check for tooling installed
	@./tools/check-tools.sh

install: check ## install luaf to the system
	@go install ./cmd/luaf

clean: uninstall
uninstall: ## install luaf to the system
	@rm -f "$(shell which luaf)"

repl: ## run luaf repl
	@go run ./cmd/luaf

test: test-go test-lua lint ## Run all tests
test-go: # Run only go tests
	@go test -cover ./...

test-lua: # Run tests interpreting lua
	@go run ./cmd/luaf ./test/all.lua

benchmark: install
	@go test -bench=.
	time luaf ./test/fib.lua

lint: lint-vet lint-ci lint-staticcheck ## Run full linting rules
lint-vet:
	@go vet ./...
lint-ci:
	@golangci-lint run
lint-staticcheck:
	@staticcheck ./...
