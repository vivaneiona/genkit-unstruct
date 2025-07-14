# Groups Example

This example demonstrates the new group functionality in genkit-unstruct that allows you to define reusable prompt and model configurations.

## Overview

The group functionality allows you to:

1. Define named groups with specific prompts and models using `WithGroup()`
2. Reference groups in struct tags using `unstruct:"group/group-name"`
3. Mix group references with direct prompt/model specifications
4. Reduce API calls by batching fields that share the same group

## Usage

### Define Groups

```go
unstruct.WithGroup("group-name", "prompt-name", "model-name")
```

### Reference Groups in Struct Tags

```go
type Person struct {
    Name string `json:"name" unstruct:"group/basic-info"`
    Age  int    `json:"age"  unstruct:"group/basic-info"`
    City string `json:"city" unstruct:"group/basic-info"`
}
```

### Mixed Usage

```go
type DetailedPerson struct {
    Name    string `json:"name" unstruct:"group/basic-info"`      // uses group
    Age     int    `json:"age"  unstruct:"group/basic-info"`       // uses group
    Address string `json:"address" unstruct:"address,gemini-1.5-pro"` // direct specification
    Email   string `json:"email" unstruct:"group/contact-info"`   // different group
}
```

## Running the Example

```bash
go run main.go
```

This will show:
1. Simple group usage with all fields in one group
2. Mixed groups and direct tags creating multiple prompt calls  
3. Detailed explanation showing the execution plan

## Benefits

- **Consistency**: Ensure related fields use the same prompt and model
- **Maintainability**: Change prompt/model for a group in one place
- **Efficiency**: Fields in the same group are batched into a single API call
- **Flexibility**: Mix groups with direct specifications as needed
