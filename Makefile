.PHONY: build test vet lint

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...
	markdownlint README.md config.example.toml || true
