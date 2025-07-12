[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

# Run the full build suite
all:
    tidy vet test build

# Run Go tests
test:
    go test -v ./...

# Build the module
build:
    go build ./...

# Build the example binary
example:
    go build -o bin/example cmd/example/main.go

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

# Format code
fmt:
    go fmt ./...

# Download modules
deps:
    go mod download


# Run 
run:
    go run main.go