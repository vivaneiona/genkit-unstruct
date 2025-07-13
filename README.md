# genkit-unstract

[![Go Report Card](https://goreportcard.com/badge/github.com/vivaneiona/genkit-unstruct)](https://goreportcard.com/report/github.com/vivaneiona/genkit-unstruct)
[![GoDoc](https://godoc.org/github.com/vivaneiona/genkit-unstruct?status.svg)](https://godoc.org/github.com/vivaneiona/genkit-unstruct)

A Go library for extracting structured data from unstructured text using AI models. Built on Google's Genkit framework, it batches prompt-based extractions, runs them concurrently, and merges JSON fragments into Go structs.

## Problem Statement

**What problem does this solve?**

Traditional text parsing with regex and string manipulation is brittle and time-consuming. When you need to extract structured data from natural language text, documents, or mixed content, you face several challenges:

- **Complex parsing logic**: Writing regex patterns and parsers for every text format
- **Type conversion overhead**: Converting extracted strings to proper Go types manually
- **Poor performance**: Making individual API calls to AI models for each field
- **Maintenance burden**: Updating parsers when text formats change

**genkit-unstract solves this by:**

1. **Automatic extraction**: Define your Go struct, specify prompts via tags, get typed data
2. **Intelligent batching**: Groups fields by prompt to minimize expensive AI API calls  
3. **Concurrent processing**: Extracts different data types in parallel for speed
4. **Type safety**: Direct conversion to Go structs with compile-time guarantees

**Example transformation:**
```go
text := "John Doe is 25 years old and lives in New York"

// Just define your structure:
type Person struct {
    Name string `json:"name" unstruct:"basic"`
    Age  int    `json:"age" unstruct:"basic"`
    City string `json:"city" unstruct:"basic"`
}

// Get typed data automatically:
person, err := u.UnstructFromText(ctx, text)
// Result: Person{Name: "John Doe", Age: 25, City: "New York"}
```

## Features

- **Intelligent Batching**: Groups fields with the same prompt for efficient processing
- **Model-Specific Extraction**: Use different AI models (Gemini Pro, Flash, etc.) for different fields
- **Nested Structures**: Extract hierarchical data with deep nesting and inheritance
- **Concurrent Processing**: Multiple extractions run in parallel with configurable concurrency
- **Type Safety**: Strongly-typed Go structs with compile-time guarantees
- **Template Support**: Support for Stick templates and custom prompt providers
- **Multi-Modal Support**: Extract structured data from text, images, and mixed content via Asset interface
- **Cost Estimation**: Built-in cost estimation and token counting
- **Retry Logic**: Configurable retry mechanisms with exponential backoff

## API Overview

The library provides two main approaches for extraction:

### Asset-Based API (Recommended)
```go
// Main extraction method - supports multiple input types
func (x *Unstructor[T]) Unstruct(ctx context.Context, assets []Asset, optFns ...func(*Options)) (*T, error)

// Create different asset types
unstruct.NewTextAsset(content string) *TextAsset
unstruct.NewImageAsset(data []byte, mimeType string) *ImageAsset  
unstruct.NewMultiModalAsset(text string, media ...*Part) *MultiModalAsset
```

### Convenience Methods
```go
// For simple text extraction (backward compatibility)
func (x *Unstructor[T]) UnstructFromText(ctx context.Context, document string, optFns ...func(*Options)) (*T, error)

// For cost estimation and planning
func (x *Unstructor[T]) DryRun(ctx context.Context, assets []Asset, optFns ...func(*Options)) (*ExecutionStats, error)
func (x *Unstructor[T]) DryRunFromText(ctx context.Context, document string, optFns ...func(*Options)) (*ExecutionStats, error)
```

## Nested Structures

The library supports nested structures with model-specific extraction:

```go
type Project struct {
    // Group fields with the same prompt for batching
    ProjectColor string  `json:"projectColor" unstruct:"project"`
    ProjectMode  string  `json:"projectMode" unstruct:"project"`
    ProjectName  string  `json:"projectName" unstruct:"project"`
    
    CertIssuer   string  `json:"certIssuer" unstruct:"cert"`
    
    // Fields without unstruct tag will cause an error unless WithFallbackPrompt() is used
    Latitude     float64 `json:"lat" unstruct:"coords"`    // explicit prompt required
    Longitude    float64 `json:"lon" unstruct:"coords"`    // explicit prompt required

    // Nested structure with model-specific extraction
    Participant struct {
        Name    string `json:"name" unstruct:"participant,gemini-1.5-pro"`
        Address string `json:"address" unstruct:"participant,gemini-1.5-pro"`
    } `json:"participant"`

    // Complex structures with custom prompts and models
    Company    Company   `unstruct:"company-info,gemini-1.5-pro"`
    Affiliated []Company `unstruct:"company-info,gemini-1.5-pro"`
}

// To handle fields without prompts, use WithFallbackPrompt option:
result, err := u.UnstructFromText(ctx, text, 
    unstruct.WithFallbackPrompt("extract-general"), // explicit fallback required
    unstruct.WithModel("gemini-1.5-flash"),
)
```

### Tag Syntax

The `unstruct` tag supports flexible syntax for controlling extraction:

- `unstruct:"prompt"` - Use a specific prompt template
- `unstruct:"gemini-1.5-pro"` - Use a specific model (inherits parent prompt)
- `unstruct:"prompt,gemini-1.5-pro"` - Use both custom prompt and model
- No tag - **ERROR**: All fields must specify a prompt or use `WithFallbackPrompt()`

### Field Grouping

Fields with the same prompt are automatically batched into a single API call for efficiency:
- `projectColor`, `projectMode`, `projectName` → 1 request with "project" prompt
- `participant.name`, `participant.address` → 1 request with "participant" prompt using "gemini-1.5-pro" model
- `certIssuer` → 1 request with "cert" prompt
- `lat`, `lon` → 1 request with "coords" prompt

## Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/vivaneiona/genkit-unstruct"
    "google.golang.org/genai"
)

type Person struct {
    Name string `json:"name" unstruct:"basic"`
    Age  int    `json:"age" unstruct:"basic"`
}

func main() {
    ctx := context.Background()
    
    // Initialize Genai client
    client, err := genai.NewClient(ctx, genai.WithAPIKey("your-api-key"))
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Initialize unstructor with client and prompt provider
    u := unstruct.New[Person](client, prompts)
    
    // Method 1: Extract from text (convenience method)
    person, err := u.UnstructFromText(ctx, "John Doe is 25 years old")
    if err != nil {
        panic(err)
    }
    
    // Method 2: Extract using Asset interface (supports multiple inputs)
    assets := []unstruct.Asset{
        unstruct.NewTextAsset("John Doe is 25 years old"),
    }
    person2, err := u.Unstruct(ctx, assets)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("%+v\n", person)
}
```

### Multi-Modal Usage

The Asset interface enables multi-modal extraction from text, images, and mixed content:

```go
// Extract from image
imageData := readImageFile("document.png") // your image loading code
assets := []unstruct.Asset{
    unstruct.NewImageAsset(imageData, "image/png"),
}
result, err := u.Unstruct(ctx, assets)

// Extract from mixed content (text + image)
assets := []unstruct.Asset{
    unstruct.NewMultiModalAsset("Extract data from this document:", 
        unstruct.NewImagePart(imageData, "image/png")),
}
result, err := u.Unstruct(ctx, assets)

// Extract from multiple text documents
assets := []unstruct.Asset{
    unstruct.NewTextAsset("First document content"),
    unstruct.NewTextAsset("Second document content"),
}
result, err := u.Unstruct(ctx, assets)
```

### Error Handling for Missing Prompts

```go
type BadStruct struct {
    Name string `json:"name" unstruct:"basic"`
    Age  int    `json:"age"` // ERROR: no prompt specified
}

// This will fail
result, err := u.UnstructFromText(ctx, "John Doe is 25 years old")
if err != nil {
    // Error: "no prompt specified for field 'age' and no fallback prompt provided"
}

// This will succeed with explicit fallback
result, err := u.UnstructFromText(ctx, "John Doe is 25 years old", 
    unstruct.WithFallbackPrompt("extract-all"))
```
```

### Running Examples

```bash
# Basic example with simple prompts
cd examples/basic && go run main.go

# Advanced example with Stick templates  
cd examples/stick && go run main.go

# Complex nested structures with model-specific extraction
cd examples/complex && go run main.go

# Cost estimation and planning
cd examples/plan && go run main.go

# Vision-based extraction with Genkit Files API
cd examples/vision && go run main.go
```

## Installation

```bash
go get github.com/vivaneiona/genkit-unstruct
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```
