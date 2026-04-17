.PHONY: run build test lint tidy fmt

run:
	go run ./cmd/api

build:
	go build -ldflags="-s -w -X main.version=$(shell git rev-parse --short HEAD 2>/dev/null || echo dev)" -o ./bin/api ./cmd/api

test:
	go test ./... -race -count=1

lint:
	go vet ./...
	gofmt -l -s . | grep -v '^$$' && exit 1 || true

tidy:
	go mod tidy

fmt:
	gofmt -w -s .
