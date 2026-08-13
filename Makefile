.PHONY: migrate seed test run setup

migrate:
	go run ./cmd/migrate up

seed:
	go run ./cmd/seed

test:
	go test ./...

run:
	go run ./cmd/server

setup: migrate seed

check: test