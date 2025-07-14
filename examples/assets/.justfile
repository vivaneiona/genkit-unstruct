# Enhanced Assets Example - File Upload Demo

[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

# Run the enhanced assets example with file upload
run:
    @echo "🚀 Running Enhanced Assets Example"
    go run main.go

# Show demo information and run if API key is set
demo:
    @echo "📋 Enhanced Assets Example Demo"
    ./demo.sh

# Build the example
build:
    @echo "🔨 Building assets example"
    go build -o assets main.go

# Clean build artifacts
clean:
    @echo "🧹 Cleaning build artifacts"
    rm -f assets
    go clean

# Show sample documents
docs:
    @echo "📄 Sample markdown documents:"
    @ls -la docs/
    @echo
    @echo "Content preview:"
    @head -n 3 docs/*.md

# Show Stick templates
templates:
    @echo "📝 Stick templates:"
    @ls -la templates/
    @echo
    @echo "Template content:"
    @head -n 3 templates/*.twig

# Test compilation
test-build:
    @echo "🧪 Testing compilation"
    go build

# Run Go tests
test:
    go test -v

# Vet code
vet:
    go vet

# Tidy dependencies  
tidy:
    go mod tidy

# Full development cycle
all: tidy vet test build