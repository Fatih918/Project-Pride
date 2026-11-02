BUF ?= buf
GO ?= go

.PHONY: proto-format proto-lint generate test vet verify

proto-format:
	$(BUF) format -w proto

proto-lint:
	$(BUF) lint

generate:
	$(BUF) generate

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

verify: proto-lint test vet
