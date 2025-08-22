SHELL := /bin/bash

.PHONY: migrate-up migrate-down migrate-create build-api-service build-user-service build-feed-service build-scheduler-service build-all run-api-service run-user-service run-feed-service run-scheduler-service

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
build-api-service:
	go build -o bin/api-service ./cmd/api-service

build-user-service:
	go build -o bin/user-service ./cmd/user-service

build-feed-service:
	go build -o bin/feed-service ./cmd/feed-service

build-scheduler-service:
	go build -o bin/scheduler-service ./cmd/scheduler-service

build-all: build-api-service build-user-service build-feed-service build-scheduler-service

# Run targets
run-api-service:
	go run ./cmd/api-service

run-user-service:
	go run ./cmd/user-service

run-feed-service:
	go run ./cmd/feed-service

run-scheduler-service:
	go run ./cmd/scheduler-service

