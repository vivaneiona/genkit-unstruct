package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// Business document structure with model selection per field type
type ExtractionRequest struct {
	Organisation struct {
		// Basic information - uses fast model
		Name         string `json:"name"`    // inherited from struct tag
		DocumentType string `json:"docType"` // inherited from struct tag

		// Financial data - uses precise model
		Revenue float64 `json:"revenue" unstruct:"prompt/financial/model/gemini-1.5-pro"`
		Budget  float64 `json:"budget" unstruct:"prompt/financial/model/gemini-1.5-pro"`

		// Complex nested data - uses most capable model with parameters
		Contact struct {
			Name  string `json:"name"`  // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
			Email string `json:"email"` // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
			Phone string `json:"phone"` // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
		} `json:"contact" unstruct:"prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40"` // Query parameters example

		// Array extraction with different model
		Projects []Project `json:"projects" unstruct:"prompt/projects/model/gemini-1.5-flash"`
	} `json:"organisation" unstruct:"prompt/basic/model/gemini-1.5-flash"` // Inherited by nested fields
}

type Project struct {
	Name   string  `json:"name"`
	Status string  `json:"status"`
	Budget float64 `json:"budget"`
}

func main() {
	ctx := context.Background()

	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: GEMINI_API_KEY environment variable is required")
		fmt.Println("Please set it with your Google AI API key:")
		fmt.Println("export GEMINI_API_KEY=your_api_key_here")
		os.Exit(1)
	}

	// Setup client
	fmt.Println("Enhanced Assets Example with URL-style Syntax")
	fmt.Println("Creating Google GenAI client...")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Prompt templates with improved instructions for nested JSON structure
	prompts := unstruct.SimplePromptProvider{
		"basic":     "Extract basic company information from the document. Return JSON with this exact structure: {\"organisation\": {\"name\": \"company_name\", \"docType\": \"document_type\"}}. Fields: {{.Keys}}. Document: {{.Document}}",
		"financial": "Extract financial data from the document. Return JSON with this exact structure: {\"organisation\": {\"revenue\": 123456, \"budget\": 789012}}. Use only numeric values without currency symbols. Fields: {{.Keys}}. Document: {{.Document}}",
		"contact":   "Extract contact information from the document. Return JSON with this exact structure: {\"organisation\": {\"contact\": {\"name\": \"contact_name\", \"email\": \"email@domain.com\", \"phone\": \"+1-555-0123\"}}}. Fields: {{.Keys}}. Document: {{.Document}}",
		"projects":  "Extract project information from the document. Return JSON with this exact structure: {\"organisation\": {\"projects\": [{\"name\": \"project_name\", \"status\": \"status\", \"budget\": 123456}]}}. Use only numeric values for budget. Fields: {{.Keys}}. Document: {{.Document}}",
	}

	// Create extractor
	extractor := unstruct.New[ExtractionRequest](client, prompts)

	// Example 1: Text document extraction
	fmt.Println("\n=== Text Document Example ===")
	runTextExample(ctx, extractor)

	// Example 2: File upload examples
	fmt.Println("\n=== File Upload Examples ===")
	runFileUploadExamples(ctx, client, extractor)

	// Example 3: Rich explain with parameters
	fmt.Println("\n=== Rich Explain Example ===")
	runExplainExample(ctx, extractor)

	// Example 4: Dry run for cost estimation
	fmt.Println("\n=== Dry Run Example ===")
	runDryRunExample(ctx, extractor)
}

func runTextExample(ctx context.Context, extractor *unstruct.Unstructor[ExtractionRequest]) {
	textDoc := `TechCorp Inc. Annual Report 2024
	
	Company: TechCorp Inc.
	Document Type: Annual Report
	Revenue: $2,500,000
	Budget: $3,000,000
	
	Contact Information:
	Name: John Smith
	Email: john@techcorp.com
	Phone: +1-555-0123
	
	Projects:
	1. Project Alpha - Status: Active - Budget: $500,000
	2. Project Beta - Status: Planning - Budget: $800,000`

	assets := []unstruct.Asset{
		unstruct.NewTextAsset(textDoc),
	}

	result, err := extractor.Unstruct(ctx, assets,
		unstruct.WithModel("gemini-1.5-flash"),
		unstruct.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Printf("Error extracting from text: %v", err)
		return
	}

	fmt.Printf("Organisation: %s (Type: %s)\n", result.Organisation.Name, result.Organisation.DocumentType)
	fmt.Printf("Financials: Revenue $%.2f, Budget $%.2f\n", result.Organisation.Revenue, result.Organisation.Budget)
	fmt.Printf("Contact: %s (%s)\n", result.Organisation.Contact.Name, result.Organisation.Contact.Email)
	fmt.Printf("Projects: %d found\n", len(result.Organisation.Projects))
	for i, proj := range result.Organisation.Projects {
		fmt.Printf("  Project %d: %s (%s) - $%.2f\n", i+1, proj.Name, proj.Status, proj.Budget)
	}
}

func runFileUploadExamples(ctx context.Context, client *genai.Client, extractor *unstruct.Unstructor[ExtractionRequest]) {
	// Find markdown files in the docs directory
	markdownFiles := findMarkdownFiles("docs")
	if len(markdownFiles) == 0 {
		fmt.Println("No markdown files found in docs/ directory")
		return
	}

	for _, filePath := range markdownFiles {
		fmt.Printf("\n--- Processing: %s ---\n", filepath.Base(filePath))

		// Create FileAsset that will upload the file to Files API
		fileAsset := unstruct.NewFileAsset(
			client,
			filePath,
			unstruct.WithDisplayName(fmt.Sprintf("Document Analysis - %s", filepath.Base(filePath))),
		)

		assets := []unstruct.Asset{fileAsset}

		result, err := extractor.Unstruct(
			ctx,
			assets,
			unstruct.WithModel("gemini-1.5-pro"), // Use Pro model for file analysis
			unstruct.WithTimeout(30*time.Second),
		)
		if err != nil {
			log.Printf("Error extracting from file %s: %v", filePath, err)
			continue
		}

		// Display results
		fmt.Printf("Organisation: %s (Type: %s)\n", result.Organisation.Name, result.Organisation.DocumentType)
		fmt.Printf("Financials: Revenue $%.2f, Budget $%.2f\n", result.Organisation.Revenue, result.Organisation.Budget)
		fmt.Printf("Contact: %s (%s)\n", result.Organisation.Contact.Name, result.Organisation.Contact.Email)
		fmt.Printf("Projects: %d found\n", len(result.Organisation.Projects))
		for i, proj := range result.Organisation.Projects {
			fmt.Printf("  Project %d: %s (%s) - $%.2f\n", i+1, proj.Name, proj.Status, proj.Budget)
		}
	}
}

func runExplainExample(ctx context.Context, extractor *unstruct.Unstructor[ExtractionRequest]) {
	// Sample document for explain demonstration
	sampleDoc := `TechCorp Inc. Financial Report Q4 2024
	
	Company: TechCorp Inc.
	Document Type: Financial Report
	Revenue: $5,200,000
	Budget: $6,000,000
	
	CEO Contact:
	Name: Sarah Johnson
	Email: sarah@techcorp.com
	Phone: +1-555-0100
	
	Active Projects:
	1. DeepAI - Status: In Progress - Budget: $1,200,000
	2. CloudScale - Status: Completed - Budget: $800,000
	3. DataMine - Status: Planning - Budget: $1,500,000`

	assets := []unstruct.Asset{
		unstruct.NewTextAsset(sampleDoc),
	}

	// Generate rich explanation with parameters
	fmt.Println("Execution Plan Analysis:")
	plan, err := extractor.Explain(ctx, assets,
		unstruct.WithModel("gemini-1.5-flash"), // Default model
		unstruct.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Printf("Error generating explain: %v", err)
		return
	}

	fmt.Println(plan)

	fmt.Println("\nParameter Details:")
	fmt.Println("• basic fields (inherited): gemini-1.5-flash (default model)")
	fmt.Println("• financial fields: gemini-1.5-pro (precision for numbers)")
	fmt.Println("• contact fields: gemini-1.5-pro with temperature=0.2, topK=40 (controlled creativity)")
	fmt.Println("• projects fields: gemini-1.5-flash (fast processing for arrays)")
	fmt.Println("\nField Inheritance:")
	fmt.Println("• organisation.name & organisation.docType inherit from organisation struct tag")
	fmt.Println("• contact.name, contact.email, contact.phone inherit from contact struct tag")
	fmt.Println("• Query parameters (temperature=0.2, topK=40) applied to contact extraction")
}

func runDryRunExample(ctx context.Context, extractor *unstruct.Unstructor[ExtractionRequest]) {
	sampleDoc := "Sample document for cost estimation: Company report with financial data and contacts."
	assets := []unstruct.Asset{
		unstruct.NewTextAsset(sampleDoc),
	}

	stats, err := extractor.DryRun(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error in dry run: %v", err)
		return
	}

	fmt.Printf("Cost Estimation:\n")
	fmt.Printf("• Prompt calls: %d\n", stats.PromptCalls)
	fmt.Printf("• Input tokens: %d\n", stats.TotalInputTokens)
	fmt.Printf("• Output tokens: %d\n", stats.TotalOutputTokens)
	fmt.Printf("• Models used: %v\n", stats.ModelCalls)
}

// findMarkdownFiles looks for .md files in the specified directory
func findMarkdownFiles(dir string) []string {
	var files []string

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return files
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) == ".md" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files
}
