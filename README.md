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
type ExtractionRequest struct {
    Organisation struct {
        // Basic information - uses fast model
        Name string `json:"name"` // inherited unstruct:"prompt/basic/model/gemini-1.5-flash"
        DocumentType string `json:"docType"` // inherited unstruct:"prompt/basic/model/gemini-1.5-flash"
        
        // Financial data - uses precise model
        Revenue float64 `json:"revenue" unstruct:"prompt/financial/model/gemini-1.5-pro"`
        Budget  float64 `json:"budget" unstruct:"prompt/financial/model/gemini-1.5-pro"`
        
        // Complex nested data - uses most capable model
        Contact struct {
            Name  string `json:"name"`  // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
            Email string `json:"email"` // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
            Phone string `json:"phone"` // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
        } `json:"contact" unstruct:"prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40"` // Query parameters example
        
        // Array extraction
        Projects []Project `json:"projects" unstruct:"prompt/projects/model/gemini-1.5-pro"` // URL syntax
    } `json:"organisation" unstruct:"prompt/basic/model/gemini-1.5-flash"` // Inherited by nested fields
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
        "basic":     "Extract basic info: {{.Keys}}. Return JSON with exact field structure.",
        "financial": "Find financial data ({{.Keys}}). Return numeric values only (e.g., 2500000 for $2.5M). Use exact JSON structure.",
        "contact":   "Extract contact details ({{.Keys}}). Return JSON with exact field structure.",
        "projects":  "List all projects with {{.Keys}}. Return budget as numeric values only (e.g., 500000 for $500K). Use exact JSON structure.",
    }
    
    // Create extractor
    extractor := unstruct.New[ExtractionRequest](client, prompts)
    
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
    fmt.Printf("Organisation: %s (Type: %s)\n", result.Organisation.Name, result.Organisation.DocumentType)
    fmt.Printf("Financials: Revenue $%.2f, Budget $%.2f\n", result.Organisation.Revenue, result.Organisation.Budget)
    fmt.Printf("Contact: %s (%s)\n", result.Organisation.Contact.Name, result.Organisation.Contact.Email)
    fmt.Printf("Projects: %d found\n", len(result.Organisation.Projects))
}
```

**Process flow:** The library:
1. Groups fields by prompt: `basic` (2 fields), `financial` (2 fields), `contact` (3 fields), `projects` (1 field)
2. Makes 4 concurrent API calls instead of 8 individual ones
3. Uses different models optimized for each data type
4. Processes multiple content types (text, PDF, image) simultaneously
5. Automatically includes asset content (files, images, text) in AI messages
6. Merges JSON fragments into a strongly-typed struct

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
    Name    string `json:"name" unstruct:"prompt/basic"`
    Age     int    `json:"age" unstruct:"prompt/basic"`
    City    string `json:"city" unstruct:"prompt/basic"`
    
    // This field requires a separate API call with different model
    Summary string `json:"summary" unstruct:"prompt/analysis/model/gpt-4"`
}
```

### Tag syntax

```go
unstruct:"prompt/basic"                          // Use named prompt with default model
unstruct:"model/gemini-1.5-flash"               // Use default prompt with override model
unstruct:"prompt/extract/model/gemini-1.5-pro"  // URL-style syntax with both prompt and model
unstruct:"group/team-info"                       // Use named group (configured via WithGroup)
```

#### Query Parameters

URL-style tags support query parameters for model configuration:

```go
unstruct:"model/gemini-1.5-flash?temperature=0.2"                    // Set temperature
unstruct:"model/gemini-1.5-flash?temperature=0.5&topK=10"            // Multiple parameters
unstruct:"prompt/extract/model/gemini-1.5-pro?topP=0.8&maxOutputTokens=1000" // Full syntax
```

**Supported parameters:**
- `temperature` (float 0.0-2.0): Controls randomness in output
- `topK` (integer): Limits token selection to top-K candidates  
- `topP` (float 0.0-1.0): Nucleus sampling threshold
- `maxOutputTokens` (integer): Maximum tokens in response

Parameters are validated and will return errors for invalid values.

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
    unstruct.WithModelFor("gemini-1.5-pro", Customer{}, "Summary"), // Per-field models
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

**Key features:**

* **Automatic batching:** Fields with the same tag are extracted together in one LLM request. This reduces the number of API calls (saves time and cost). No need to manually orchestrate which fields go in which prompt – the tags handle that.
* **Parallel execution:** It runs all the needed LLM calls concurrently under the hood (uses Go routines and an `errgroup` by default), so overall extraction is as fast as possible given your model choices.
* **Multi-model support:** You can assign different models to different field groups. For example, use a quick model for simple text and a more advanced model for numbers or summaries. The library will route each group to the right model automatically.
* **Multi-modal inputs:** Supports extracting from plain text, images, and PDFs in one shot. You just provide an array of `unstruct.Asset` (there are helpers like `NewTextAsset`, `NewFileAsset`, `NewImageAsset` etc.), and it handles packaging that content for the model (e.g. uploading files, embedding images).
* **Structured output:** You get a typed Go struct out of the process. genkit-unstruct takes care of parsing the model’s JSON output and merging multiple responses. Nested structs and slices are supported (nested fields can inherit their parent’s prompt tag by default), so you can model complex data hierarchies.
* **Extras for optimization:** There’s a `DryRun()` mode that simulates the extraction to estimate how many tokens *would* be used and how many calls *would* be made – useful for cost planning. You can also `Explain()` an extraction which prints an execution plan (which prompt groups will run, with what model, etc.). These features helped us sanity-check prompts and budget before running big jobs.
* **Extensible design:** The concurrency is abstracted via a small `Runner` interface. By default it just uses a background errgroup, but you can swap in a custom runner – in our case we built a Temporal-compatible runner so we could run the extraction inside a Temporal workflow (to get durable, replayable execution). The prompt system is also pluggable: you can supply prompt templates via a simple map (as shown) or use the provided Twig template integration for more complex prompt generation. The idea is to integrate into real production workflows, not be a toy.

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

Extract these specific fields: {{ KeyList }}

Return as JSON with exact field names.
```

**Template variables available:**
- `{{ Keys }}` - Array of field names for iteration: `{% for key in Keys %}{{ key }}{% endfor %}`
- `{{ KeyList }}` - Comma-separated string of field names: `"name, age, email"`
- `{{ Document }}` - The document content when using text assets
- `{{ Version }}` - Template version number
- `{{ Tag }}` - Template tag name

**Note:** When using assets (files, images, text), the document content is automatically added to the AI message. You don't need to include `Document: {{ Document }}` in your templates - the assets are passed directly to the model alongside your prompt.

```go
// Use Twig template engine
prompts, _ := unstruct.NewStickPromptProvider(
    unstruct.WithFS(os.DirFS("."), "templates"),
)
```

### Nested structures

```go
type Company struct {
    Name string `json:"name" unstruct:"prompt/company"`
    
    // Nested struct with field-specific extraction rules
    CEO struct {
        Name  string `json:"name" unstruct:"prompt/person/model/gemini-1.5-pro"`
        Email string `json:"email" unstruct:"prompt/person/model/gemini-1.5-pro"`
    } `json:"ceo"`
    
    // Array extraction
    Employees []Employee `json:"employees" unstruct:"prompt/team/model/gemini-1.5-flash"`
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
🚀 Running Enhanced Assets Example
go run main.go
Enhanced Assets Example with URL-style Syntax
Creating Google GenAI client...

=== Text Document Example ===
2025/07/19 05:57:51 INFO Extraction completed successfully type=main.ExtractionRequest
Organisation: TechCorp Inc. (Type: Annual Report)
Financials: Revenue $2500000.00, Budget $3000000.00
Contact: John Smith (john@techcorp.com)
Projects: 2 found
  Project 1: Project Alpha (Active) - $500000.00
  Project 2: Project Beta (Planning) - $800000.00

=== File Upload Examples ===

--- Processing: meeting-minutes.md ---
2025/07/19 05:57:56 INFO Extraction completed successfully type=main.ExtractionRequest
Organisation:  (Type: Meeting Minutes)
Financials: Revenue $2300000.00, Budget $800000.00
Contact:  ()
Projects: 9 found
  Project 1: Migrate to microservices architecture (Complete) - $800000.00
  Project 2: Implement new authentication system (Complete) - $800000.00
  Project 3: Reduce page load times by 30% (Complete) - $800000.00
  Project 4: Launch mobile application (iOS/Android) (Complete) - $200000.00
  Project 5: Beta test AI recommendation engine (Complete) - $200000.00
  Project 6: Integrate with 5 new third-party APIs (Complete) - $200000.00
  Project 7: Rebrand company visual identity (Complete) - $120000.00
  Project 8: Launch content marketing campaign (Complete) - $120000.00
  Project 9: Attend 4 industry conferences (Complete) - $120000.00

--- Processing: product-requirements.md ---
2025/07/19 05:58:00 INFO Extraction completed successfully type=main.ExtractionRequest
Organisation: EduTech Solutions Inc. (Type: Product Requirements Document)
Financials: Revenue $500000.00, Budget $2000000.00
Contact:  ()
Projects: 1 found
  Project 1: SmartLearn Educational Platform (Draft) - $1200000.00

--- Processing: tech-spec.md ---
2025/07/19 05:58:04 INFO Extraction completed successfully type=main.ExtractionRequest
Organisation: TechCorp Inc (Type: Technical Specification)
Financials: Revenue $0.00, Budget $500000.00
Contact: John Doe (john.doe@company.com)
Projects: 1 found
  Project 1: Advanced AI Development Platform (High) - $500000.00

=== Rich Explain Example ===
Execution Plan Analysis:
2025/07/19 05:58:04 INFO Dry run completed prompt_calls=4 total_input_tokens=708 total_output_tokens=245 models_used=2
Unstructor Execution Plan (estimated costs)
SchemaAnalysis (cost=24.6, tokens(in=10), fields=[organisation.revenue organisation.budget organisation.contact.name organisation.contact.email organisation.contact.phone organisation.projects.name organisation.projects.status organisation.projects.budget organisation.name organisation.docType])
  ├─ PromptCall "financial" (model=gemini-1.5-pro, cost=4.7, tokens(in=171,out=54), fields=[organisation.revenue organisation.budget])
  ├─ PromptCall "contact" (model=gemini-1.5-pro, cost=4.8, tokens(in=183,out=71), fields=[organisation.contact.name organisation.contact.email organisation.contact.phone])
  ├─ PromptCall "projects" (model=gemini-1.5-flash, cost=4.9, tokens(in=190,out=71), fields=[organisation.projects.name organisation.projects.status organisation.projects.budget])
  ├─ PromptCall "basic" (model=gemini-1.5-flash, cost=4.6, tokens(in=164,out=49), fields=[organisation.name organisation.docType])
  └─ MergeFragments (cost=1.5, fields=[organisation.revenue organisation.budget organisation.contact.name organisation.contact.email organisation.contact.phone organisation.projects.name organisation.projects.status organisation.projects.budget organisation.name organisation.docType])


Parameter Details:
• basic fields (inherited): gemini-1.5-flash (default model)
• financial fields: gemini-1.5-pro (precision for numbers)
• contact fields: gemini-1.5-pro with temperature=0.2, topK=40 (controlled creativity)
• projects fields: gemini-1.5-flash (fast processing for arrays)

Field Inheritance:
• organisation.name & organisation.docType inherit from organisation struct tag
• contact.name, contact.email, contact.phone inherit from contact struct tag
• Query parameters (temperature=0.2, topK=40) applied to contact extraction

=== Dry Run Example ===
2025/07/19 05:58:04 INFO Dry run completed prompt_calls=4 total_input_tokens=389 total_output_tokens=245 models_used=2
Cost Estimation:
• Prompt calls: 4
• Input tokens: 389
• Output tokens: 245
• Models used: map[gemini-1.5-flash:2 gemini-1.5-pro:2]
```

```bash
➜  genkit-unstract git:(main) ✗ just do openai_and_gemini run
go run main.go
Extraction Example
==============================
Demonstrating different Gemini models with varied parameters
Note: For OpenAI integration, see the openai example

Creating Google GenAI client...

=== Company Analysis ===
2025/07/19 05:20:39 Error extracting company data: merge "financial": financials: json: cannot unmarshal string into Go struct field .growth_rate of type float64

=== Model Selection Strategy ===
2025/07/19 05:20:39 INFO Dry run completed prompt_calls=9 total_input_tokens=1093 total_output_tokens=774 models_used=3
Model Distribution and Execution Plan:
Unstructor Execution Plan (estimated costs)
SchemaAnalysis (cost=50.6, tokens(in=10), fields=[technology.security.compliance technology.security.encryption contact.ceo.name contact.ceo.email contact.investor_relations.name contact.investor_relations.email contact.investor_relations.phone contact.press.name contact.press.email competitive.market_share competitive.competitive_rank competitive.key_differentiators competitive.market_trends financials.revenue financials.profit financials.market_cap financials.growth_rate financials.risk_score financials.outlook technology.primary_tech technology.cloud_provider technology.architecture strategy.market_position strategy.competitors strategy.strengths strategy.threats strategy.opportunities strategy.strategic_priority company.name company.industry company.founded company.headquarters])
  ├─ PromptCall "technical" (model=gemini-1.5-pro, cost=4.2, tokens(in=117,out=54), fields=[technology.security.compliance technology.security.encryption])
  ├─ PromptCall "contact" (model=gemini-1.5-pro, cost=4.1, tokens(in=109,out=49), fields=[contact.ceo.name contact.ceo.email])
  ├─ PromptCall "contact" (model=gemini-1.5-pro, cost=4.2, tokens(in=125,out=71), fields=[contact.investor_relations.name contact.investor_relations.email contact.investor_relations.phone])
  ├─ PromptCall "contact" (model=gemini-1.5-pro, cost=4.1, tokens(in=110,out=49), fields=[contact.press.name contact.press.email])
  ├─ PromptCall "competitive" (model=gemini-2.0-flash-exp, cost=4.2, tokens(in=117,out=98), fields=[competitive.market_share competitive.competitive_rank competitive.key_differentiators competitive.market_trends])
  ├─ PromptCall "financial" (model=gemini-1.5-pro, cost=4.4, tokens(in=142,out=142), fields=[financials.revenue financials.profit financials.market_cap financials.growth_rate financials.risk_score financials.outlook])
  ├─ PromptCall "technical" (model=gemini-1.5-pro, cost=4.2, tokens(in=120,out=76), fields=[technology.primary_tech technology.cloud_provider technology.architecture])
  ├─ PromptCall "strategy" (model=gemini-1.5-pro, cost=4.4, tokens(in=141,out=142), fields=[strategy.market_position strategy.competitors strategy.strengths strategy.threats strategy.opportunities strategy.strategic_priority])
  ├─ PromptCall "basic" (model=gemini-1.5-flash, cost=4.1, tokens(in=112,out=93), fields=[company.name company.industry company.founded company.headquarters])
  └─ MergeFragments (cost=3.7, fields=[technology.security.compliance technology.security.encryption contact.ceo.name contact.ceo.email contact.investor_relations.name contact.investor_relations.email contact.investor_relations.phone contact.press.name contact.press.email competitive.market_share competitive.competitive_rank competitive.key_differentiators competitive.market_trends financials.revenue financials.profit financials.market_cap financials.growth_rate financials.risk_score financials.outlook technology.primary_tech technology.cloud_provider technology.architecture strategy.market_position strategy.competitors strategy.strengths strategy.threats strategy.opportunities strategy.strategic_priority company.name company.industry company.founded company.headquarters])


Model Selection Strategy:
� Gemini Flash      → Basic facts (optimize for speed & cost)
� Gemini Pro (T=0.1) → Financial data (optimize for precision)
� Gemini Pro        → Technical analysis (domain knowledge)
🎨 Gemini Pro (T=0.3) → Strategic insights (creative reasoning)
� Gemini Pro (T=0.1) → Contact extraction (maximize accuracy)
🧠 Gemini 2.0 Flash  → Competitive analysis (advanced reasoning)

Parameter Tuning:
• Temperature 0.1: High precision for financial/contact data
• Temperature 0.3: Balanced creativity for strategic analysis
• TopK=5: Focused selection for contact accuracy
• TopK=40: Broader selection for strategic creativity

=== Performance and Cost Analysis ===
2025/07/19 05:20:39 INFO Dry run completed prompt_calls=9 total_input_tokens=1055 total_output_tokens=774 models_used=3
Performance Analysis:
• Total API calls: 9
• Input tokens: 1055
• Output tokens: 774
• Models used: map[gemini-1.5-flash:1 gemini-1.5-pro:7 gemini-2.0-flash-exp:1]

Optimization Strategy:
• Use Flash models for simple extraction tasks
• Reserve Pro models for complex analysis
• Tune temperature based on task requirements
• Group related fields to minimize API calls
• Consider using experimental models for cutting-edge features

Future Integration:
• OpenAI models could be integrated for specific reasoning tasks
• Claude models for creative writing and analysis
• Anthropic models for ethical reasoning and safety
• Mixed provider strategy for cost and capability optimization
```