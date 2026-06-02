.PHONY: build ci fmt fmt-check test vet

build:
	go build -o bin/mcp-data-gateway ./cmd/mcp-data-gateway

fmt:
	gofmt -w ./cmd ./internal

fmt-check:
	@files=$$(gofmt -l ./cmd ./internal); \
	if [ -n "$$files" ]; then \
		printf 'gofmt needed:\n%s\n' "$$files"; \
		exit 1; \
	fi

test:
	go test ./...

vet:
	go vet ./...

ci: fmt-check vet test build
