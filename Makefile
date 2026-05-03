.PHONY: migrate

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

migrate:
	go run ./cmd/migrator $(ARGS)