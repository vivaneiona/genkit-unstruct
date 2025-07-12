# ⚙︎ vivaneiona/genkit-unstract

genkit-unstract is a Go library for extracting structured data from unstructured text. It runs multiple prompt-based extractions concurrently and merges the JSON fragments into a strongly-typed Go struct, giving you reliable, type-safe results at scale.

## Features

- **Multi-prompt extraction**: Automatically batch fields with the same prompt for efficient processing
- **Model-specific extraction**: Use different AI models for different fields or nested structures
- **Nested structure support**: Extract complex hierarchical data with deep nesting
- **Concurrent processing**: Multiple extractions run in parallel for optimal performance
- **Type-safe results**: Strongly-typed Go structs with compile-time safety
- **Flexible templates**: Support for Stick templates and custom prompt providers

## Complex Nested Structures

The library supports sophisticated nested structures with model-specific extraction:

```go
type Project struct {
    // Group fields with the same prompt for batching
    ProjectColor string  `json:"projectColor" unstruct:"project"`
    ProjectMode  string  `json:"projectMode" unstruct:"project"`
    ProjectName  string  `json:"projectName" unstruct:"project"`
    
    CertIssuer   string  `json:"certIssuer" unstruct:"cert"`
    Latitude     float64 `json:"lat"`    // default extraction
    Longitude    float64 `json:"lon"`    // default extraction

    // Nested structure with model-specific extraction
    Participant struct {
        Name    string `json:"name" unstruct:"gemini-1.5-pro"`
        Address string `json:"address" unstruct:"gemini-1.5-pro"`
    } `json:"participant"`

    // Complex structures with custom prompts and models
    Company    Company   `unstruct:"prompt-name,gemini-1.5-pro"`
    Affiliated []Company `unstruct:"prompt-name,gemini-1.5-pro"`
}
```

### Tag Syntax

The `unstruct` tag supports flexible syntax for controlling extraction:

- `unstruct:"prompt"` - Use a specific prompt template
- `unstruct:"gemini-1.5-pro"` - Use a specific model (inherits parent prompt)
- `unstruct:"prompt,gemini-1.5-pro"` - Use both custom prompt and model
- No tag - Uses default prompt and model

### Field Grouping

Fields with the same prompt are automatically batched into a single API call for efficiency:
- `projectColor`, `projectMode`, `projectName` → 1 request with "project" prompt
- `participant.name`, `participant.address` → 1 request with "gemini-1.5-pro" model
- `certIssuer` → 1 request with "cert" prompt
- `lat`, `lon` → 1 request with default prompt

## Examples

```bash
# Basic example with simple prompts
just do basic run

# Advanced example with Stick templates
just do stick run

# Complex nested structures with model-specific extraction
just do complex run
```
