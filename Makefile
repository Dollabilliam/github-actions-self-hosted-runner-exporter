.PHONY: fmt vet test build run healthz

CONFIG ?= $(if $(wildcard .config.local.json),.config.local.json,config.json)
HEALTHZ_URL ?= http://127.0.0.1:9176/healthz

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...

test:
	go test ./...

build:
	go build -o bin/github-actions-self-hosted-runner-exporter ./cmd/github-actions-self-hosted-runner-exporter

run:
	go run ./cmd/github-actions-self-hosted-runner-exporter -config $(CONFIG)

healthz:
	curl -fsS $(HEALTHZ_URL)
