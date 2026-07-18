GO ?= go
SMOKE_BINARY := ./tmp/elefante-phase-1

.PHONY: test test-race test-ddev-integration vet smoke verify

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-ddev-integration:
	ELEFANTE_DDEV_INTEGRATION=1 $(GO) test -tags=integration ./internal/providers/ddev -run TestDDEVIntegration -count=1

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
