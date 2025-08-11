SHELL := /bin/bash

.PHONY: migrate-up migrate-down migrate-create

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


