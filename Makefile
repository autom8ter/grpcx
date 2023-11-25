.PHONY: help generate test

help:
	@echo "generate - generate code"
	@echo "test     - run tests"
generate:
	@go generate ./...


test:
	@go test -coverprofile=coverage.out ./...
