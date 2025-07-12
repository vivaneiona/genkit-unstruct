# genkit-unstract

[![Go Report Card](https://goreportcard.com/badge/github.com/vivaneiona/genkit-unstruct)](https://goreportcard.com/report/github.com/vivaneiona/genkit-unstruct)
[![GoDoc](https://godoc.org/github.com/vivaneiona/genkit-unstruct?status.svg)](https://godoc.org/github.com/vivaneiona/genkit-unstruct)

A Go library for extracting structured data from unstructured text using AI models. Built on Google's Genkit framework, it batches prompt-based extractions, runs them concurrently, and merges JSON fragments into Go structs.

## Features

- **Intelligent Batching**: Groups fields with the same prompt for efficient processing
- **Model-Specific Extraction**: Use different AI models (Gemini Pro, Flash, etc.) for different fields
- **Nested Structures**: Extract hierarchical data with deep nesting and inheritance
- **Concurrent Processing**: Multiple extractions run in parallel with configurable concurrency
- **Type Safety**: Strongly-typed Go structs with compile-time guarantees
- **Template Support**: Support for Stick templates and custom prompt providers
- **Vision Support**: Extract structured data from images using Genkit Files API
- **Cost Estimation**: Built-in cost estimation and token counting
- **Retry Logic**: Configurable retry mechanisms with exponential backoff

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
result, err := u.Unstruct(ctx, text, 
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
)

type Person struct {
    Name string `json:"name" unstruct:"basic"`
    Age  int    `json:"age" unstruct:"basic"`
}

func main() {
    ctx := context.Background()
    
    // Initialize unstructor with prompt provider
    u := unstruct.New(prompts)
    
    // Extract data from text
    person, err := u.Unstruct(ctx, "John Doe is 25 years old")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("%+v\n", person)
}
```

### Error Handling for Missing Prompts

```go
type BadStruct struct {
    Name string `json:"name" unstruct:"basic"`
    Age  int    `json:"age"` // ERROR: no prompt specified
}

// This will fail
result, err := u.Unstruct(ctx, "John Doe is 25 years old")
if err != nil {
    // Error: "no prompt specified for field 'age' and no fallback prompt provided"
}

// This will succeed with explicit fallback
result, err := u.Unstruct(ctx, "John Doe is 25 years old", 
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
