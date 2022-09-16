default: test build
.PHONY: test

test:
	@go test -v -race ./...

build:
	@go build