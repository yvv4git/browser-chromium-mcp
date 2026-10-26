.PHONY: build test vet lint

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

lint:
	markdownlint README.md config.example.toml || true
