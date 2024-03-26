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

.PHONY: hash
## hash: update static files hashes in index.html
hash:
	sed -i '' -E "s/styles\.css\?crc=[0-9a-z]+/styles.css?crc=$$(crc32 ./pages/styles.css)/" pages/index.html
	sed -i '' -E "s/script\.js\?crc=[0-9a-z]+/script.js?crc=$$(crc32 ./pages/script.js)/" pages/index.html

.PHONY: dev
## dev: run the wrangler pages dev
dev: hash
	wrangler pages dev pages/

.PHONY: build the WASM binary
build:
	tinygo build -o ./pages/main.wasm -target wasm -no-debug main.go
