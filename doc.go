// Package unstruct provides intelligent extraction of structured data from
// unstructured text, images, and documents using AI models. Built on Google's Genkit framework,
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
//   - File handling complexity: Converting documents to text before processing
//
// The unstruct package solves this by providing:
//
//   - Automatic extraction: Define your Go struct, specify prompts via tags, get typed data
//   - Intelligent batching: Groups fields by prompt to minimize expensive AI API calls
//   - Concurrent processing: Extracts different data types in parallel for speed
//   - Type safety: Direct conversion to Go structs with compile-time guarantees
//   - Multi-modal support: Direct processing of PDFs, Word docs, images, and text
//
// # Basic Usage
//
// Define a struct with unstruct tags and extract data automatically:
//
//	type ProjectInfo struct {
//	    ProjectCode string `json:"projectCode" unstruct:"prompt/project/model/gemini-1.5-flash"`
//	    ProjectName string `json:"projectName" unstruct:"prompt/project/model/gemini-1.5-flash"`
//	    Budget      string `json:"budget" unstruct:"prompt/financial/model/gemini-1.5-pro"`
//	    Currency    string `json:"currency" unstruct:"prompt/financial/model/gemini-1.5-pro"`
//	}
//
//	func main() {
//	    ctx := context.Background()
//	    client, _ := genai.NewClient(ctx, &genai.ClientConfig{
//	        Backend: genai.BackendGeminiAPI,
//	        APIKey:  os.Getenv("GEMINI_API_KEY"),
//	    })
//	    defer client.Close()
//
//	    prompts := unstruct.SimplePromptProvider{
//	        "project":    "Extract project info: {{.Keys}}. Return JSON with exact field structure.",
//	        "financial":  "Find financial data ({{.Keys}}). Return numeric values only.",
//	    }
//
//	    u := unstruct.New[ProjectInfo](client, prompts)
//	    project, err := u.Unstruct(ctx, []unstruct.Asset{
//	        unstruct.NewTextAsset("Project Alpha with code ABC-123, budget $500,000 USD"),
//	    })
//	    // Result: ProjectInfo{ProjectCode: "ABC-123", ProjectName: "Project Alpha", Budget: "500000", Currency: "USD"}
//	}
//
// # Asset-Based API
//
// The package supports multi-modal extraction through the Asset interface:
//
//	// Single text extraction
//	assets := []unstruct.Asset{
//	    unstruct.NewTextAsset("Project Alpha with code ABC-123, budget $500,000 USD"),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
//	// Image extraction (charts, diagrams, documents)
//	imageData := readImageFile("project-chart.png")
//	assets := []unstruct.Asset{
//	    unstruct.NewImageAsset(imageData, "image/png"),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
//	// Multi-modal: text instructions + image content
//	assets := []unstruct.Asset{
//	    unstruct.NewMultiModalAsset("Extract project data from this financial chart:",
//	        unstruct.NewImagePart(imageData, "image/png")),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
//	// File extraction (PDFs, Word docs, etc.)
//	assets := []unstruct.Asset{
//	    unstruct.NewFileAsset(client, "project-requirements.pdf"),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
//	// Batch file processing for multiple documents
//	filePaths := []string{"meeting-notes.md", "tech-spec.pdf", "user-requirements.docx"}
//	assets := []unstruct.Asset{
//	    unstruct.NewBatchFileAsset(client, filePaths),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
//	// Combined extraction from multiple sources
//	assets := []unstruct.Asset{
//	    unstruct.NewTextAsset("Extract project data from these documents:"),
//	    unstruct.NewFileAsset(client, "requirements.pdf"),
//	    unstruct.NewImageAsset(chartImageData, "image/png"),
//	}
//	result, err := u.Unstruct(ctx, assets)
//
// # File Processing
//
// The package provides robust file processing capabilities through the Google Files API.
// Files are automatically uploaded and processed by AI models that can analyze various
// document formats including PDFs, Word documents, text files, and more.
//
//	// Single file processing with configuration
//	fileAsset := unstruct.NewFileAsset(client, "project-requirements.pdf")
//	fileAsset.DisplayName = "Project Requirements Document"
//	fileAsset.AutoCleanup = true  // Clean up after processing
//	fileAsset.IncludeMetadata = true  // Include file size, checksum, etc.
//	result, err := u.Unstruct(ctx, []unstruct.Asset{fileAsset})
//
//	// Batch file processing with progress tracking
//	filePaths := []string{
//	    "meeting-notes.md",
//	    "technical-spec.pdf",
//	    "user-requirements.docx",
//	}
//
//	batchAsset := unstruct.NewBatchFileAsset(client, filePaths,
//	    unstruct.WithBatchProgressCallback(func(processed, total int, currentFile string) {
//	        fmt.Printf("Processing %d/%d: %s\n", processed, total, currentFile)
//	    }),
//	    unstruct.WithBatchAutoCleanup(true),
//	)
//	result, err := u.Unstruct(ctx, []unstruct.Asset{batchAsset})
//
// Files are uploaded to Google's Files API where they become available to Gemini models
// for content analysis. The AI can read and extract structured data from file contents,
// not just metadata. All file uploads are automatically cleaned up after processing
// when AutoCleanup is enabled.
//
// # Nested Structures and Complex Extraction
//
// The library supports complex nested structures with model-specific extraction,
// based on real-world business document processing:
//
//	type ExtractionRequest struct {
//	    Organisation struct {
//	        // Basic information - uses fast model
//	        Name         string `json:"name"`    // inherits from parent unstruct tag
//	        DocumentType string `json:"docType"` // inherits from parent unstruct tag
//
//	        // Financial data - uses precise model for accuracy
//	        Revenue float64 `json:"revenue" unstruct:"prompt/financial/model/gemini-1.5-pro"`
//	        Budget  float64 `json:"budget" unstruct:"prompt/financial/model/gemini-1.5-pro"`
//
//	        // Complex nested data with model parameters
//	        Contact struct {
//	            Name  string `json:"name"`  // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
//	            Email string `json:"email"` // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
//	            Phone string `json:"phone"` // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
//	        } `json:"contact" unstruct:"prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40"`
//
//	        // Array extraction
//	        Projects []Project `json:"projects" unstruct:"prompt/projects/model/gemini-1.5-flash"`
//	    } `json:"organisation" unstruct:"prompt/basic/model/gemini-1.5-flash"` // Inherited by nested fields
//	}
//
//	type Project struct {
//	    Name   string  `json:"name"`
//	    Status string  `json:"status"`
//	    Budget float64 `json:"budget"`
//	}
//
// # Tag Syntax
//
// The unstruct tag uses a flexible URL-style syntax for controlling extraction:
//
//   - unstruct:"prompt/basic" - Use a specific prompt template
//   - unstruct:"model/gemini-1.5-pro" - Use default prompt with specific model
//   - unstruct:"prompt/financial/model/gemini-1.5-pro" - Custom prompt and model
//   - unstruct:"prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40" - With parameters
//   - No tag - ERROR: All fields must specify a prompt or use WithFallbackPrompt()
//
// The URL-style syntax supports complex model names and query parameters:
//
//	type ProjectData struct {
//	    Name      string  `unstruct:"prompt/project/model/gemini-1.5-flash"`
//	    Budget    float64 `unstruct:"prompt/financial/model/gemini-1.5-pro?temperature=0.1"`
//	    Contact   string  `unstruct:"model/gemini-1.5-pro"`  // Uses default prompt
//	    Timeline  string  `unstruct:"prompt/timeline"`       // Uses default model
//	}
//
// Nested structures inherit parent tags, allowing efficient field grouping:
//
//	type Organisation struct {
//	    Contact struct {
//	        Name  string `json:"name"`  // Inherits parent tag
//	        Email string `json:"email"` // Inherits parent tag
//	    } `unstruct:"prompt/contact/model/gemini-1.5-pro"`
//	} `unstruct:"prompt/basic/model/gemini-1.5-flash"` // Default for all nested fields
//
// # Field Grouping and Batching
//
// Fields with the same prompt are automatically batched into a single API call
// for efficiency. This significantly reduces costs and improves performance:
//
//	// These fields will be processed in one API call:
//	type ProjectInfo struct {
//	    ProjectCode string `unstruct:"prompt/project/model/gemini-1.5-flash"`
//	    ProjectName string `unstruct:"prompt/project/model/gemini-1.5-flash"`
//	    Status      string `unstruct:"prompt/project/model/gemini-1.5-flash"`
//	    Priority    string `unstruct:"prompt/project/model/gemini-1.5-flash"`
//
//	    // These financial fields will be processed in a separate API call:
//	    Budget   string `unstruct:"prompt/financial/model/gemini-1.5-pro"`
//	    Currency string `unstruct:"prompt/financial/model/gemini-1.5-pro"`
//	}
//
// # Configuration Options
//
// The package provides various configuration options for fine-tuning extraction:
//
//	result, err := u.Unstruct(ctx, assets,
//	    unstruct.WithModel("gemini-1.5-flash"),               // Default model override
//	    unstruct.WithFallbackPrompt("extract-general"),       // Handle fields without prompts
//	    unstruct.WithTimeout(30*time.Second),                 // Set timeout
//	    unstruct.WithRetry(3, 2*time.Second),                // Configure retries with backoff
//	    unstruct.WithConcurrency(5),                          // Control parallelism
//	)
//
// # Cost Estimation and Performance Monitoring
//
// The package includes built-in cost estimation and token counting for budget planning:
//
//	stats, err := u.DryRun(ctx, assets)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Estimated cost: $%.4f\n", stats.EstimatedCost)
//	fmt.Printf("Input tokens: %d\n", stats.InputTokens)
//	fmt.Printf("Output tokens: %d\n", stats.OutputTokens)
//	fmt.Printf("Groups: %d\n", len(stats.Groups))
//	for _, group := range stats.Groups {
//	    fmt.Printf("  Group %s: %d fields, model %s\n",
//	        group.Prompt, len(group.Keys), group.Model)
//	}
//
// # Error Handling
//
// Fields without unstruct tags require explicit handling:
//
//	type BadStruct struct {
//	    Name string `json:"name" unstruct:"prompt/basic"`
//	    Age  int    `json:"age"` // ERROR: no prompt specified
//	}
//
//	// This will fail
//	result, err := u.Unstruct(ctx, assets)
//	if err != nil {
//	    // Error: field 'Age' has no prompt specified
//	}
//
//	// This will succeed with explicit fallback
//	result, err := u.Unstruct(ctx, assets,
//	    unstruct.WithFallbackPrompt("extract-all"))
//
// # Performance Features
//
//   - Intelligent batching: Groups fields by prompt to minimize API calls
//   - Concurrent processing: Multiple extractions run in parallel
//   - Configurable concurrency: Control the number of parallel requests
//   - Retry logic: Configurable retry mechanisms with exponential backoff
//   - Cost optimization: Built-in cost estimation and token counting
//   - Model selection: Use different models optimized for different data types
//
// # Multi-Modal Support
//
// The Asset interface enables extraction from various input types:
//
//   - Text documents via TextAsset
//   - Images via ImageAsset (PNG, JPEG, charts, diagrams, etc.)
//   - Files via FileAsset (PDFs, Word docs, text files, etc.)
//   - Batch file processing via BatchFileAsset
//   - Mixed content via MultiModalAsset
//   - Multiple documents in a single extraction
//
// File assets are automatically uploaded to Google's Files API and processed
// by Gemini models that can analyze document content, not just metadata.
// Supported file types include PDFs, Word documents, text files, markdown,
// and other formats supported by Gemini models.
//
// # Template Support
//
// The package supports flexible prompt templating through the PromptProvider
// interface, including simple templates and custom providers:
//
//	// Simple template provider
//	prompts := unstruct.SimplePromptProvider{
//	    "project":    "Extract project info: {{.Keys}}. Return JSON with exact field structure.",
//	    "financial":  "Find financial data ({{.Keys}}). Return numeric values only.",
//	    "timeline":   "Extract dates and timeline info: {{.Keys}}. Use ISO format.",
//	}
//
//	// Custom provider with versioning and context
//	type CustomPrompts struct{}
//	func (p CustomPrompts) GetPrompt(tag string, version int) (string, error) {
//	    // Load from database, file system, or external service
//	    return loadPromptFromDatabase(tag, version)
//	}
//
// For more examples and detailed usage, see the examples/ directory and
// the project documentation at https://github.com/vivaneiona/genkit-unstruct
package unstruct
