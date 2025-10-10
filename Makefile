SHELL := /bin/bash

# Ensure Go-installed binaries (protoc plugins) are on PATH
GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

.PHONY: migrate-up migrate-down migrate-create build-api-service build-user-service build-feed-service build-scheduler-service build-ai-service build-all run-api-service run-user-service run-feed-service run-scheduler-service run-ai-service test infra-up infra-down proto-tools generate

migrate-up:
	go run ./cmd/migrator up

migrate-down:
	go run ./cmd/migrator down

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=<name>"; exit 1; fi
	@dir=db/migrations; \
	 mkdir -p $$dir; \
	 next=$$(printf "%06d" $$(($$(ls $$dir/*.up.sql 2>/dev/null | wc -l) + 1))); \
	 up="$$dir/$$next_$(NAME).up.sql"; \
	 down="$$dir/$$next_$(NAME).down.sql"; \
	 touch $$up $$down; \
	 echo "created $$up and $$down"

# Build targets
build-web:
	npm --prefix web install
	npm --prefix web run build

build-api-service: build-web
	@echo "--> Copying frontend assets for embedding..."
	@cp -r web/build cmd/api-service/dist
	@echo "--> Building api-service binary..."
	go build -o bin/api-service ./cmd/api-service
	@echo "--> Cleaning up temporary frontend assets..."
	@rm -rf cmd/api-service/dist

build-user-service:
	go build -o bin/user-service ./cmd/user-service

build-feed-service:
	go build -o bin/feed-service ./cmd/feed-service

build-scheduler-service:
	go build -o bin/scheduler-service ./cmd/scheduler-service

build-ai-service:
	go build -o bin/ai-service ./cmd/ai-service

build-all: build-api-service build-user-service build-feed-service build-scheduler-service build-ai-service

# Run targets
run-api-service:
	go run ./cmd/api-service

run-user-service:
	go run ./cmd/user-service

run-feed-service:
	go run ./cmd/feed-service

run-scheduler-service:
	go run ./cmd/scheduler-service

run-ai-service:
	go run ./cmd/ai-service

test:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | tail -n 1

infra-up:
	docker compose up -d postgres redis kafka

infra-down:
	docker compose down


# Proto tools and code generation
proto-tools:
	@command -v protoc >/dev/null 2>&1 || { echo "ERROR: protoc not found. Install it first (e.g., 'brew install protobuf')."; exit 1; }
	@echo "--> Ensuring protoc Go plugins are installed..."
	@GOBIN=$(GOBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	@GOBIN=$(GOBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

generate: proto-tools
	@echo "--> Generating Go code from proto files..."
	@protoc -I proto \
		--go_out=. --go_opt=module=github.com/Fancu1/phoenix-rss \
		--go-grpc_out=. --go-grpc_opt=module=github.com/Fancu1/phoenix-rss \
		proto/feed.proto proto/user.proto
	@protoc -I proto \
		--go_out=. \
		proto/article_events.proto
	@echo "--> Proto generation complete."

