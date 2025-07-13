// Package unstruct provides intelligent extraction of structured data from
// unstructured text using AI models. Built on Google's Genkit framework,
// it automatically batches prompt-based extractions, runs them concurrently,
// and merges JSON fragments into strongly-typed Go structs.
//
// # Problem Statement
//
// Traditional text parsing with regex and string manipulation is brittle and
// time-consuming. When extracting structured data from natural language text,
// documents, or mixed content, you face several challenges:
//
//   - Complex parsing logic: Writing regex patterns and parsers for every text format
//   - Type conversion overhead: Converting extracted strings to proper Go types manually
//   - Poor performance: Making individual API calls to AI models for each field
//   - Maintenance burden: Updating parsers when text formats change
//
// The unstruct package solves this by providing:
//
//   - Automatic extraction: Define your Go struct, specify prompts via tags, get typed data
//   - Intelligent batching: Groups fields by prompt to minimize expensive AI API calls
//   - Concurrent processing: Extracts different data types in parallel for speed
//   - Type safety: Direct conversion to Go structs with compile-time guarantees
//
// # Basic Usage
//
// Define a struct with unstruct tags and extract data automatically:
//
//	type Person struct {
//	    Name string `json:"name" unstruct:"basic"`
//	    Age  int    `json:"age" unstruct:"basic"`
//	    City string `json:"city" unstruct:"basic"`
//	}
//
//	func main() {
//	    ctx := context.Background()
//	    client, _ := genai.NewClient(ctx, genai.WithAPIKey("your-api-key"))
//	    u := unstruct.New[Person](client, prompts)
//
//	    person, err := u.UnstructFromText(ctx, "John Doe is 25 years old and lives in New York")
//	    // Result: Person{Name: "John Doe", Age: 25, City: "New York"}
//	}
//
// # Asset-Based API
//
// The package supports multi-modal extraction through the Asset interface:
//
//	// Text extraction
//	assets := []unstruct.Asset{
//	    unstruct.NewTextAsset("John Doe is 25 years old"),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
//	// Image extraction
//	imageData := readImageFile("document.png")
//	assets := []unstruct.Asset{
//	    unstruct.NewImageAsset(imageData, "image/png"),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
//	// Mixed content (text + image)
//	assets := []unstruct.Asset{
//	    unstruct.NewMultiModalAsset("Extract data from this document:",
//	        unstruct.NewImagePart(imageData, "image/png")),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
// # Nested Structures
//
// The library supports complex nested structures with model-specific extraction:
//
//	type Project struct {
//	    // Group fields with the same prompt for batching
//	    ProjectColor string  `json:"projectColor" unstruct:"project"`
//	    ProjectMode  string  `json:"projectMode" unstruct:"project"`
//	    ProjectName  string  `json:"projectName" unstruct:"project"`
//
//	    // Nested structure with model-specific extraction
//	    Participant struct {
//	        Name    string `json:"name" unstruct:"participant,gemini-1.5-pro"`
//	        Address string `json:"address" unstruct:"participant,gemini-1.5-pro"`
//	    } `json:"participant"`
//
//	    // Complex structures with custom prompts and models
//	    Company    Company   `unstruct:"company-info,gemini-1.5-pro"`
//	    Affiliated []Company `unstruct:"company-info,gemini-1.5-pro"`
//	}
//
// # Tag Syntax
//
// The unstruct tag supports flexible syntax for controlling extraction:
//
//   - unstruct:"prompt" - Use a specific prompt template
//   - unstruct:"gemini-1.5-pro" - Use a specific model (inherits parent prompt)
//   - unstruct:"prompt,gemini-1.5-pro" - Use both custom prompt and model
//   - No tag - ERROR: All fields must specify a prompt or use WithFallbackPrompt()
//
// # Field Grouping and Batching
//
// Fields with the same prompt are automatically batched into a single API call
// for efficiency. This significantly reduces costs and improves performance:
//
//	// These fields will be processed in one API call:
//	ProjectColor string `unstruct:"project"`
//	ProjectMode  string `unstruct:"project"`
//	ProjectName  string `unstruct:"project"`
//
// # Configuration Options
//
// The package provides various configuration options:
//
//	result, err := u.UnstructFromText(ctx, text,
//	    unstruct.WithModel("gemini-1.5-flash"),           // Specify model
//	    unstruct.WithFallbackPrompt("extract-general"),   // Handle fields without prompts
//	    unstruct.WithTimeout(30*time.Second),             // Set timeout
//	    unstruct.WithMaxRetries(3),                       // Configure retries
//	    unstruct.WithConcurrency(5),                      // Control parallelism
//	)
//
// # Cost Estimation
//
// The package includes built-in cost estimation and token counting:
//
//	stats, err := u.DryRunFromText(ctx, document)
//	fmt.Printf("Estimated cost: $%.4f\n", stats.EstimatedCost)
//	fmt.Printf("Input tokens: %d\n", stats.InputTokens)
//	fmt.Printf("Groups: %d\n", len(stats.Groups))
//
// # Error Handling
//
// Fields without unstruct tags require explicit handling:
//
//	type BadStruct struct {
//	    Name string `json:"name" unstruct:"basic"`
//	    Age  int    `json:"age"` // ERROR: no prompt specified
//	}
//
//	// This will fail
//	result, err := u.UnstructFromText(ctx, "John Doe is 25 years old")
//
//	// This will succeed with explicit fallback
//	result, err := u.UnstructFromText(ctx, "John Doe is 25 years old",
//	    unstruct.WithFallbackPrompt("extract-all"))
//
// # Performance Features
//
//   - Intelligent batching: Groups fields by prompt to minimize API calls
//   - Concurrent processing: Multiple extractions run in parallel
//   - Configurable concurrency: Control the number of parallel requests
//   - Retry logic: Configurable retry mechanisms with exponential backoff
//   - Cost optimization: Built-in cost estimation and token counting
//
// # Multi-Modal Support
//
// The Asset interface enables extraction from various input types:
//
//   - Text documents via TextAsset
//   - Images via ImageAsset (PNG, JPEG, etc.)
//   - Mixed content via MultiModalAsset
//   - Multiple documents in a single extraction
//
// # Template Support
//
// The package supports flexible prompt templating through the PromptProvider
// interface, including Stick templates and custom prompt providers with
// context-aware variable substitution.
//
// For more examples and detailed usage, see the examples/ directory and
// the project documentation at https://github.com/vivaneiona/genkit-unstruct
package unstruct
