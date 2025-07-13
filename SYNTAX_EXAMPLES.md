# New Unstruct Tag Syntax

The unstruct library now supports a flexible syntax for specifying prompts and models in struct tags. All model names are accepted regardless of provider.

## Supported Formats

### 1. Explicit Prompt and Model
```go
type MyStruct struct {
    Field string `json:"field" unstruct:"my_prompt,my_model"`
}
```
- Explicitly specifies both prompt and model
- Works with any model name (no restrictions)

### 2. Model-Only with Inheritance
```go
type MyStruct struct {
    Field string `json:"field" unstruct:"model/googleai/gemini-1.5-pro"`
}
```
- Uses `model/` prefix to specify only the model
- Inherits prompt from parent struct or uses default
- Supports any provider prefix (googleai/, vertex/, anthropic/, openai/, etc.)

### 3. Prompt-Only
```go
type MyStruct struct {
    Field string `json:"field" unstruct:"prompt/custom_extraction"`
}
```
- Uses `prompt/` prefix to specify only the prompt
- No model specified (uses default from Options)

### 4. Prompt-Only (Traditional)
```go
type MyStruct struct {
    Field string `json:"field" unstruct:"my_prompt"`
}
```
- Single value is treated as prompt name
- No model specified (uses default from Options)

### 5. Empty Tag (Inheritance)
```go
type MyStruct struct {
    Field string `json:"field" unstruct:""`
}
```
- Inherits both prompt and model from parent context

## Complete Example

```go
type DocumentExtraction struct {
    // Traditional explicit format
    Title string `json:"title" unstruct:"document_info,googleai/gemini-1.5-pro"`
    
    // Model-only with inheritance (inherits prompt from parent)
    Author string `json:"author" unstruct:"model/anthropic/claude-3-sonnet"`
    
    // Prompt-only (uses default model)
    Summary string `json:"summary" unstruct:"prompt/content_summary"`
    
    // Traditional prompt (uses default model)
    Keywords string `json:"keywords" unstruct:"keyword_extraction"`
    
    // Nested structure with inherited context
    Company struct {
        Name    string `json:"name"`    // Inherits from parent
        Address string `json:"address"` // Inherits from parent
    } `json:"company" unstruct:"company_info,vertex/gemini-1.5-flash"`
    
    // Provider-specific models
    PersonalInfo struct {
        Name  string `json:"name" unstruct:"model/openai/gpt-4"`
        Email string `json:"email" unstruct:"model/custom-model-v2"`
    } `json:"personal_info" unstruct:"personal_extraction"`
}
```

## Key Changes from Previous Version

1. **No "Known Model" Restrictions**: Any model name is accepted, regardless of provider
2. **Provider Prefixes Supported**: `googleai/`, `vertex/`, `anthropic/`, `openai/`, etc.
3. **New Prefix Syntax**: `model/` and `prompt/` prefixes for clarity
4. **Consistent Behavior**: Single values are always treated as prompts
5. **Backward Compatibility**: Explicit `prompt,model` format still works

## Migration Guide

### Old Syntax → New Syntax

```go
// Old: Single model name (only worked for "known" models)
Field string `unstruct:"gemini-1.5-pro"`

// New: Use model/ prefix
Field string `unstruct:"model/gemini-1.5-pro"`

// Old: Provider prefix not recognized
Field string `unstruct:"googleai/gemini-1.5-pro"` // Treated as prompt

// New: Use model/ prefix for provider-prefixed models
Field string `unstruct:"model/googleai/gemini-1.5-pro"`

// Old & New: Explicit format works the same
Field string `unstruct:"prompt_name,model_name"`
```
