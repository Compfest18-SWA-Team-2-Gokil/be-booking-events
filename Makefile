.PHONY: seed test run setup check

seed:
	go run ./cmd/seed

test:
	go test ./...

run:
	go run ./cmd/server

setup: seed

check: test