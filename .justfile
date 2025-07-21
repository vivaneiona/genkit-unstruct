[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

# Version mgmt
mod? version '.justfiles/version.just'

# `just do basic run`
mod? do 'examples/.justfile'

# Run the full build suite
all:
    just tidy
    just lint
    just test
    just build

# Verify dependencies
verify:
    go mod download
    go mod verify

# Run Go tests
test:
    go test -v -race -coverprofile=coverage.out ./...

# Build the module
build:
    go build ./...


# Remove build artifacts
clean:
    rm -rf bin/
    go clean ./...

# Tidy and vendor dependencies
tidy:
    go mod tidy
    go mod vendor

# Static analysis
vet:
    go vet ./...

# Comprehensive linting (includes vet, fmt check, and additional checks)
lint:
    @echo "Running comprehensive linting..."
    @echo "  ├─ go vet (static analysis)"
    @go vet ./...
    @echo "  ├─ go fmt (formatting check)"
    @test -z "$(gofmt -l . | grep -v vendor)" || (echo "Files need formatting:" && gofmt -l . | grep -v vendor && echo "Run 'just fmt' to fix." && exit 1)
    @echo "  ├─ golangci-lint (static analysis)"
    @golangci-lint run
    @echo "  ├─ go mod verify (dependency integrity)"
    @go mod verify
    @echo "  └─ build check (compilation)"
    @go build ./...
    @echo "All linting checks passed!"



# Format code
fmt:
    go fmt ./...

# Download modules
deps:
    go mod download
