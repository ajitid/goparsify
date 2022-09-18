default: test bench build
.PHONY: test

test:
	@go test -v -race ./...

bench:
	@go test -benchmem -bench=. ./json

build:
	@go build