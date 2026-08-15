.PHONY: seed test run setup check

seed:
	go run ./cmd/seed

test:
	go test ./...

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

setup: seed

check: test