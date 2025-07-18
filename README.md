# genkit-unstruct

Concurrent data extraction from unstructured text and images using AI models.

[![Go Report Card](https://goreportcard.com/badge/github.com/vivaneiona/genkit-unstruct)](https://goreportcard.com/report/github.com/vivaneiona/genkit-unstruct)

A Go library for extracting structured data from unstructured sources using AI models. Built on Google Genkit, it automatically batches fields by prompt, executes extractions concurrently, and merges results into typed structs.

## Example

Extract complex business data from mixed documents (invoices, contracts, reports) with different AI models optimized for each data type:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    
    unstruct "github.com/vivaneiona/genkit-unstruct"
    "google.golang.org/genai"
)

// Business document structure with model selection per field type
type BusinessExtract struct {
    // Basic information - uses fast model
    CompanyName string `json:"companyName" unstruct:"basic,gemini-1.5-flash"`
    DocumentType string `json:"docType" unstruct:"basic,gemini-1.5-flash"`
    
    // Financial data - uses precise model
    Revenue float64 `json:"revenue" unstruct:"financial,gemini-1.5-pro"`
    Budget  float64 `json:"budget" unstruct:"financial,gemini-1.5-pro"`
    
    // Complex nested data - uses most capable model
    Contact struct {
        Name  string `json:"name" unstruct:"contact,gemini-2.0-pro"`
        Email string `json:"email" unstruct:"contact,gemini-2.0-pro"`
        Phone string `json:"phone" unstruct:"contact,gemini-2.0-pro"`
    } `json:"contact"`
    
    // Array extraction
    Projects []Project `json:"projects" unstruct:"projects,gemini-1.5-pro"`
}

type Project struct {
    Name   string  `json:"name"`
    Status string  `json:"status"`
    Budget float64 `json:"budget"`
}

func main() {
    ctx := context.Background()
    
    // Setup client
    client, _ := genai.NewClient(ctx, &genai.ClientConfig{
        Backend: genai.BackendGeminiAPI,
        APIKey:  os.Getenv("GEMINI_API_KEY"),
    })
    defer client.Close()
    
    // Prompt templates (alternatively use Twig templates)
    prompts := unstruct.SimplePromptProvider{
        "basic":     "Extract basic info: {{.Keys}} from: {{.Document}}",
        "financial": "Find financial data ({{.Keys}}) in: {{.Document}}",
        "contact":   "Extract contact details ({{.Keys}}) from: {{.Document}}",
        "projects":  "List all projects with {{.Keys}} from: {{.Document}}",
    }
    
    // Create extractor
    extractor := unstruct.New[BusinessExtract](client, prompts)
    
    // Multi-modal extraction from various sources
    assets := []unstruct.Asset{
        unstruct.NewTextAsset("TechCorp Inc. Annual Report 2024..."),
        unstruct.NewFileAsset(client, "contract.pdf"),        // PDF upload
        // unstruct.NewImageAsset(imageData, "image/png"),       // Image analysis
    }
    
    // Extract with configuration options
    result, err := extractor.Unstruct(ctx, assets,
        unstruct.WithModel("gemini-1.5-flash"),               // Default model
        unstruct.WithTimeout(30*time.Second),                 // Timeout
        unstruct.WithRetry(3, 2*time.Second),                // Retry logic
    )
    
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Extracted data:\n")
    fmt.Printf("Company: %s (Type: %s)\n", result.CompanyName, result.DocumentType)
    fmt.Printf("Financials: Revenue $%.2f, Budget $%.2f\n", result.Revenue, result.Budget)
    fmt.Printf("Contact: %s (%s)\n", result.Contact.Name, result.Contact.Email)
    fmt.Printf("Projects: %d found\n", len(result.Projects))
}
```

**Process flow:** The library:
1. Groups fields by prompt: `basic` (2 fields), `financial` (2 fields), `contact` (3 fields), `projects` (1 field)
2. Makes 4 concurrent API calls instead of 8 individual ones
3. Uses different models optimized for each data type
4. Processes multiple content types (text, PDF, image) simultaneously
5. Merges JSON fragments into a strongly-typed struct

## Installation

```bash
go get github.com/vivaneiona/genkit-unstruct@latest
```

## Core concepts

### Field grouping
Fields with the same `unstruct` tag are automatically batched into a single AI call:

```go
type Customer struct {
    // These fields will be processed in a single API call
    Name    string `json:"name" unstruct:"basic"`
    Age     int    `json:"age" unstruct:"basic"`
    City    string `json:"city" unstruct:"basic"`
    
    // This field requires a separate API call with different model
    Summary string `json:"summary" unstruct:"analysis,gpt-4"`
}
```

### Tag syntax

```go
unstruct:"prompt"                    // Use prompt with default model
unstruct:"prompt,gemini-1.5-pro"     // Custom prompt with specific model  
unstruct:"gemini-2.0-flash"          // Use default prompt with override model
unstruct:"group/team-info"           // Use named group (configured via WithGroup)
```

### Multi-modal assets

Process any combination of content types:

```go
assets := []unstruct.Asset{
    unstruct.NewTextAsset("Raw text content"),
    unstruct.NewImageAsset(imageBytes, "image/png"),
    unstruct.NewFileAsset(client, "document.pdf"),
    unstruct.NewMultiModalAsset("Analyze this:", 
        unstruct.NewTextPart("Description"),
        unstruct.NewImagePart(imageBytes, "image/png"),
    ),
}
```

## Configuration

Configuration options for extraction:

```go
result, err := extractor.Unstruct(ctx, assets,
    unstruct.WithModel("gemini-1.5-flash"),           // Default model
    unstruct.WithTimeout(30*time.Second),             // Request timeout
    unstruct.WithRetry(3, 1*time.Second),             // Retry config
    unstruct.WithGroup("team", "people", "gemini-pro"), // Named groups
    unstruct.WithModelFor("gemini-2.0-pro", Customer{}, "Summary"), // Per-field models
)
```

## Cost optimization

**Dry runs** estimate costs before making actual API calls:

```go
stats, err := extractor.DryRun(ctx, assets, unstruct.WithModel("gemini-1.5-pro"))
fmt.Printf("Estimated cost: %d input + %d output tokens\n", 
    stats.TotalInputTokens, stats.TotalOutputTokens)
fmt.Printf("API calls: %d\n", stats.PromptCalls)
fmt.Printf("Models used: %v\n", stats.ModelCalls)
```

**Execution plans** show exactly what will happen:

```go
plan, err := extractor.Explain(ctx, assets, unstruct.WithModel("gemini-1.5-pro"))
fmt.Println(plan)
// Output:
// Execution Plan:
// 1. prompt-group-1 (gemini-1.5-flash): [Name, Age, City] -> ~120 tokens
// 2. prompt-group-2 (gemini-1.5-pro): [Summary] -> ~200 tokens
// Total: 2 API calls, ~320 tokens
```

## Advanced features

### Prompt templates

Create reusable templates in `templates/` directory:

```twig
<!-- templates/customer.twig -->
Extract customer information from this business document.

Focus on identifying:
- Personal details (name, age, location)
- Contact information
- Business relationships

Extract these specific fields: {% for key in Keys %}{{ key }}{% if not loop.last %}, {% endif %}{% endfor %}

Return as JSON with exact field names.

Document: {{ Document }}
```

```go
// Use Twig template engine
prompts, _ := unstruct.NewStickPromptProvider(
    unstruct.WithFS(os.DirFS("."), "templates"),
)
```

### Nested structures

```go
type Company struct {
    Name string `json:"name" unstruct:"company"`
    
    // Nested struct with field-specific extraction rules
    CEO struct {
        Name  string `json:"name" unstruct:"person,gemini-1.5-pro"`
        Email string `json:"email" unstruct:"person,gemini-1.5-pro"`
    } `json:"ceo"`
    
    // Array extraction
    Employees []Employee `json:"employees" unstruct:"team,gemini-1.5-flash"`
}
```

### Concurrency control

```go
// Limit concurrent API calls
runner := unstruct.NewLimitedRunner(3)
result, err := extractor.Unstruct(ctx, assets, unstruct.WithRunner(runner))
```

## API reference

### Core methods
- `Unstruct(ctx, assets, opts...)` – Extract data from assets
- `UnstructFromText(ctx, text, opts...)` – Extract from plain text (convenience)
- `DryRun(ctx, assets, opts...)` – Estimate costs without API calls
- `Explain(ctx, assets, opts...)` – Show execution plan

### Asset builders
- `NewTextAsset(text)` – Plain text content
- `NewImageAsset(data, mimeType)` – Image analysis
- `NewFileAsset(client, path, opts...)` – File upload to Google Files API
- `NewMultiModalAsset(text, parts...)` – Mixed content
- `NewBatchFileAsset(client, paths)` – Multiple files

### Options
- `WithModel(name)` – Set default model
- `WithTimeout(duration)` – Request timeout
- `WithRetry(max, backoff)` – Retry configuration
- `WithGroup(name, prompt, model)` – Named groups
- `WithModelFor(model, type, field)` – Per-field model overrides
- `WithRunner(runner)` – Custom concurrency control

## Testing

```bash
export GEMINI_API_KEY=your_api_key
go test ./...
```

## Examples

Read .justfile and run

```bash
genkit-unstract git:(main) ✗ just do
➤ vet
➤ basic ...
➤ stick ...
➤ complex ...
➤ plan ...
➤ vision ...
➤ stats_demo ...
➤ assets ...
➤ openai ...
➤ vertexai ...
```


```bash
➜  genkit-unstract git:(main) ✗ just do assets
➤ run        # Run the enhanced assets example with file upload
➤ demo       # Show demo information and run if API key is set
➤ build      # Build the example
➤ clean      # Clean build artifacts
➤ docs       # Show sample documents
➤ templates  # Show Stick templates
➤ test-build # Test compilation
➤ test       # Run Go tests
➤ vet        # Vet code
➤ tidy       # Tidy dependencies
➤ all        # Full development cycle
```


```bash
➜  genkit-unstract git:(main) ✗ just do assets run
🚀 Running Enhanced Assets Example
go run main.go
Creating Google GenAI client...
Setting up Stick template engine...

=== Text Document Example ===
2025/07/18 18:51:03 INFO Extraction completed successfully type=main.DocumentMetadata
Title: Technical Report: Advanced AI Systems
Description: This document describes the implementation of machine learning algorithms for natural language processing.
Category: Technology Research
Author: 
Date: January 15, 2024
Version: 1.2

=== File Upload Examples ===

--- Processing: meeting-minutes.md ---
2025/07/18 18:51:08 INFO Extraction completed successfully type=main.ProjectInfo
Project Code: 
Project Name: Q1 2025 Objectives
Budget: 0.00 $
Timeline: Jan 1, 2025 to Mar 31, 2025
Status: Planning
Priority: High
Project Lead: 
Team Size: 0

--- Processing: product-requirements.md ---
2025/07/18 18:51:13 INFO Extraction completed successfully type=main.ProjectInfo
Project Code: 
Project Name: SmartLearn
Budget: 1200000.00 $
Timeline: January 10, 2025 to 
Status: Draft
Priority: 
Project Lead: Sarah Mitchell
Team Size: 12

--- Processing: tech-spec.md ---
2025/07/18 18:51:18 INFO Extraction completed successfully type=main.ProjectInfo
Project Code: AI-DEV-2024-001
Project Name: Advanced AI Development Platform
Budget: 0.00 USD
Timeline: 2024-02-01 to 2024-08-31
Status: 
Priority: High
Project Lead: Sarah Johnson
Team Size: 0

=== Dry Run Example ===
2025/07/18 18:51:18 INFO Dry run completed prompt_calls=2 total_input_tokens=238 total_output_tokens=147 models_used=1
Estimated prompt calls: 2
Estimated input tokens: 238
Estimated output tokens: 147
```
