SHELL := /bin/bash
export DATABASE_URL ?= postgres://watch:watch@localhost:5432/watch?sslmode=disable

.PHONY: run crawl crawl-dry migrate test test-integration lint build docker

run: ## run the API locally
	APP_ENV=local go run ./cmd/api

crawl: ## run a full crawl locally
	APP_ENV=local go run ./cmd/crawler $(ARGS)

crawl-dry: ## print sample listings without writing (usage: make crawl-dry ARGS="-sources=jackroad")
	APP_ENV=local go run ./cmd/crawler -dry-run -max-pages=1 $(ARGS)

migrate:
	go run ./cmd/migrate up

test:
	go vet ./... && go test -race -count=1 ./...

test-integration: ## real PostgreSQL via embedded-postgres (downloads ~30MB once)
	TEST_EMBEDDED_PG=1 go test -count=1 -run TestRepositoryEndToEnd ./internal/repository/

lint:
	gofmt -l . && go vet ./...

build:
	CGO_ENABLED=0 go build -trimpath -o bin/api ./cmd/api && CGO_ENABLED=0 go build -trimpath -o bin/crawler ./cmd/crawler

docker:
	docker build -t watch-compare-backend:local .
