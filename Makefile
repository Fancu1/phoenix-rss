SHELL := /bin/bash

.PHONY: migrate-up migrate-down migrate-create build-server build-user-service build-feed-service build-all run-server run-user-service run-feed-service

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
build-server:
	go build -o bin/server ./cmd/server

build-user-service:
	go build -o bin/user-service ./cmd/user-service

build-feed-service:
	go build -o bin/feed-service ./cmd/feed-service

build-all: build-server build-user-service build-feed-service

# Run targets
run-server:
	go run ./cmd/server

run-user-service:
	go run ./cmd/user-service

run-feed-service:
	go run ./cmd/feed-service

