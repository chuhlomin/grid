.PHONY: help
## help: print this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: vet
## vet: run the linter
vet:
	@go vet ./...

.PHONY: test
## test: run the tests
test:
	@go test ./...

.PHONY: dev
## dev: run the wrangler pages dev
dev:
	wrangler pages dev pages/

.PHONY: build the WASM binary
build:
	tinygo build -o ./pages/main.wasm -target wasm -no-debug main.go
