sad# Complex Nested Structure Extraction Example

This example demonstrates advanced features of the genkit-unstract library, including:

## Features Demonstrated

### 🎯 Nested Structure Extraction
- Multi-level data hierarchies
- Complex object relationships
- Deep nesting support

### 🤖 Model-Specific Field Processing
- Different models for different fields using `unstruct-with` tags
- Optimization based on field complexity
- Mixed model strategies

### 📋 Example Structure

```go
type ComplexProject struct {
    ProjectCode string  `json:"projectCode" unstruct:"code"`
    CertIssuer  string  `json:"certIssuer"  unstruct:"cert"`
    Latitude    float64 `json:"lat"`
    Longitude   float64 `json:"lon"`
    
    // High-accuracy participant extraction with Pro model
    Participant struct {
        Name    string `json:"name"    unstruct-with:"gemini-1.5-pro"`
        Address string `json:"address" unstruct-with:"gemini-1.5-pro"`
    } `json:"participant"`

    // Fast owner extraction with Flash model
    Owner struct {
        Name    string `json:"name"    unstruct-with:"gemini-1.5-flash"`
        Address string `json:"address" unstruct-with:"gemini-1.5-flash"`
    } `json:"owner"`
    
    // Mixed complexity requirements
    Details struct {
        Description string `json:"description" unstruct-with:"gemini-1.5-pro"`
        Budget      struct {
            Amount   float64 `json:"amount"   unstruct-with:"gemini-1.5-flash"`
            Currency string  `json:"currency" unstruct-with:"gemini-1.5-flash"`
        } `json:"budget"`
        Timeline struct {
            StartDate string `json:"startDate" unstruct-with:"gemini-1.5-pro"`
            EndDate   string `json:"endDate"   unstruct-with:"gemini-1.5-pro"`
        } `json:"timeline"`
    } `json:"details"`
}
```

## Model Selection Strategy

- **gemini-1.5-pro**: Used for complex text extraction requiring high accuracy
  - Participant names and addresses
  - Project descriptions
  - Timeline dates
  
- **gemini-1.5-flash**: Used for simple, structured data extraction
  - Owner information
  - Budget amounts and currencies
  - Basic identifiers

- **Default model**: Used for standard fields without specific requirements

## Usage

```bash
# Set your Gemini API key
export GEMINI_API_KEY=your_api_key_here

# Run the example
go run main.go
```

## Expected Output

The example processes three complex documents and extracts:
- Project metadata (codes, certificates, coordinates)
- Participant information (names, addresses)
- Owner details
- Project descriptions, budgets, and timelines

Each field is processed with its optimal model for best accuracy vs. performance balance.

## Key Benefits

1. **Intelligent Model Selection**: Different fields use optimal models
2. **Nested Structure Support**: Deep object hierarchies are handled automatically
3. **Performance Optimization**: Fast models for simple data, powerful models for complex extraction
4. **Flexible Configuration**: Easy to adjust model assignments per field
