ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: run test fmt tidy build

fmt:
	go fmt ./...

tidy:
	go mod tidy

test:
	go test ./... -v

build: tidy fmt
	go build ./...

run: build
	go run cmd/main.go