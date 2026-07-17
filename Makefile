GO ?= go
SMOKE_BINARY := ./tmp/elefante-phase-1

.PHONY: test test-race vet smoke verify

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

smoke:
	@mkdir -p ./tmp
	@set -e; \
		trap 'rm -f "$(SMOKE_BINARY)"' EXIT; \
		$(GO) build -o "$(SMOKE_BINARY)" ./cmd/elefante; \
		"$(SMOKE_BINARY)" --help >/dev/null; \
		"$(SMOKE_BINARY)" version

verify: test test-race vet smoke
