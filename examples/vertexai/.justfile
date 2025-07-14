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

# Remove build artifacts
clean:
    rm -rf bin/
    go clean ./...

# Tidy and vendor dependencies
tidy:
    go mod tidy

# Static analysis
vet:
    go vet ./...

# Format code
fmt:
    go fmt ./...

# Download modules
deps:
    go mod download

# Run the Stick template example
run:
    go run main.go

# Run with sample GEMINI_API_KEY (for demo purposes - replace with real key)
demo:
    @echo "Note: Set GEMINI_API_KEY environment variable before running"
    @echo "Example: export GEMINI_API_KEY=your_actual_api_key"
    go run main.go
