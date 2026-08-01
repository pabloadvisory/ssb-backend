.PHONY: build test test-race test-integration vet staticcheck vuln fmt check migrate-up seed-demo run docker-up docker-down

GO ?= go

build:
	$(GO) build -trimpath -o bin/ssb ./cmd/ssb

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-integration:
	docker compose run --build --rm test

vet:
	$(GO) vet ./...

staticcheck:
	$(GO) run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...

vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

check: fmt vet staticcheck test vuln

migrate-up:
	$(GO) run ./cmd/ssb migrate up

seed-demo:
	docker compose run --build --rm api seed demo

run:
	$(GO) run ./cmd/ssb serve

docker-up:
	docker compose up --build

docker-down:
	docker compose down
