# Build for dst. `make` / `make build` is the full gate:
#   fmt, ascii, vet, lint, modernize, tests, determinism.
# Toolchain + commit hook: `mise install` (pins in mise.toml; postinstall runs make install-hooks).

.PHONY: build test fmt-check vet lint modernize-check modernize ascii-check determinism install-hooks

build: fmt-check ascii-check vet lint modernize-check test determinism

install-hooks:
	@mkdir -p .git/hooks
	@ln -sfn ../../ops/scripts/githooks/pre-commit .git/hooks/pre-commit
	@chmod +x ops/scripts/githooks/pre-commit
	@echo "installed .git/hooks/pre-commit -> ops/scripts/githooks/pre-commit"

fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

lint:
	golangci-lint run ./...

modernize-check:
	go fix -diff ./...

modernize:
	go fix ./...

ascii-check:
	@ops/scripts/ascii-check.sh

test:
	go test -race ./...

# Determinism gate for the single-goroutine packages (rng, sched, clock, ...).
determinism:
	bash ops/scripts/ci-determinism.sh
