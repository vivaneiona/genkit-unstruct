# Batch File Processing Example

[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

# Run the batch file processing example
run:
    @echo "🚀 Running Batch File Processing Example"
    go run main.go

# Build the example
build:
    @echo "🔨 Building batch assets example"
    go build -o assets_batch main.go

# Clean build artifacts and sample documents
clean:
    @echo "🧹 Cleaning up"
    rm -f assets_batch
    rm -rf docs/

# Show example information
info:
    @echo "📋 Batch File Processing Example"
    @echo ""
    @echo "This example demonstrates BatchFileAsset capabilities:"
    @echo "• Process multiple files in batches"
    @echo "• Advanced progress tracking"
    @echo "• Metadata collection and statistics"
    @echo "• Auto cleanup of uploaded files"
    @echo "• Cost estimation for batch operations"
    @echo ""
    @echo "Prerequisites:"
    @echo "• GEMINI_API_KEY environment variable"
    @echo "• Sample documents (auto-created if missing)"

# Test the example with dry run
test:
    @echo "🧪 Testing batch processing with dry run"
    @echo "This will create sample documents and estimate costs"
    go run main.go

# Setup sample documents manually
setup:
    @echo "📄 Creating sample documents"
    mkdir -p docs
    @echo "Sample documents will be created automatically when you run the example"
