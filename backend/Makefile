.PHONY: build build-embed build-noembed generate wire-check test test-unit test-integration test-e2e

VERSION ?= $(shell ./scripts/resolve-version.sh)
LDFLAGS ?= -s -w -X main.Version=$(VERSION)

build: build-embed

build-embed:
	CGO_ENABLED=0 go build -tags=embed -ldflags="$(LDFLAGS)" -trimpath -o bin/server ./cmd/server

build-noembed:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o bin/server ./cmd/server

generate:
	go generate ./ent
	go generate ./cmd/server

wire-check:
	go run github.com/google/wire/cmd/wire@v0.7.0 diff ./cmd/server

test:
	go test ./...
	golangci-lint run ./...

test-unit:
	go test -tags=unit ./...

test-integration:
	go test -tags=integration ./...

test-e2e:
	bash ./scripts/e2e-test.sh

test-e2e-local:
	go test -tags=e2e -v -timeout=300s ./internal/integration/...
