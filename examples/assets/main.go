package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// ExtractionRequest demonstrates sophisticated document extraction with:
// - Template-based prompts (loaded from templates/ folder)
// - Model selection per field type for optimal cost/quality balance
// - Nested structures with field inheritance
// - Model parameters (temperature, topK) for fine-tuned control
type ExtractionRequest struct {
	Organisation struct {
		// Basic information - uses fast model for simple text extraction
		Name         string `json:"name"`    // inherited from struct tag
		DocumentType string `json:"docType"` // inherited from struct tag

		// Financial data - uses precise model for accurate number extraction
		Revenue float64 `json:"revenue" unstruct:"prompt/financial/model/gemini-1.5-pro"`
		Budget  float64 `json:"budget" unstruct:"prompt/financial/model/gemini-1.5-pro"`

		// Contact information - uses precise model with controlled parameters
		Contact struct {
			Name  string `json:"name"`  // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
			Email string `json:"email"` // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
			Phone string `json:"phone"` // Inherits prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40
		} `json:"contact" unstruct:"prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40"`

		// Project array - uses fast model for bulk processing
		Projects []Project `json:"projects" unstruct:"prompt/projects/model/gemini-1.5-flash"`
	} `json:"organisation" unstruct:"prompt/basic/model/gemini-1.5-flash"` // Root level - inherited by nested fields
}

// Project represents individual project information
type Project struct {
	Name   string  `json:"name"`
	Status string  `json:"status"`
	Budget float64 `json:"budget"`
}

func main() {
	ctx := context.Background()

	// Validate environment setup
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("▪︎ Error: GEMINI_API_KEY environment variable is required")
		fmt.Println("Please set it with your Google AI API key:")
		fmt.Println("  export GEMINI_API_KEY=your_api_key_here")
		os.Exit(1)
	}

	// Initialize Google GenAI client
	fmt.Println("⚙︎ Creating Google GenAI client...")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Fatal("▪︎ Failed to create client:", err)
	}

	// Create template-based prompt provider
	fmt.Println("⚙︎ Loading prompt templates from templates/ folder...")
	prompts, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
	)
	if err != nil {
		log.Fatal("▪︎ Failed to create prompt provider:", err)
	}

	// Initialize extractor with templates
	extractor := unstruct.New[ExtractionRequest](client, prompts)
	fmt.Println("⚙︎ Extractor initialized with template-based prompts")

	// Process documents from docs folder
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("⚙︎ GENKIT-UNSTRUCT FILE PROCESSING")
	fmt.Println(strings.Repeat("=", 60))

	runFileUploadExamples(ctx, client, extractor)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("⚙︎ Processing completed!")
	fmt.Println(strings.Repeat("=", 60))
}

func runFileUploadExamples(ctx context.Context, client *genai.Client, extractor *unstruct.Unstructor[ExtractionRequest]) {

	// Find markdown files in the docs directory
	markdownFiles := findMarkdownFiles("docs")
	if len(markdownFiles) == 0 {
		fmt.Println("▪︎ No markdown files found in docs/ directory")
		return
	}

	fmt.Printf("Found %d document(s) to process...\n", len(markdownFiles))

	for i, filePath := range markdownFiles {
		fmt.Printf("⚙︎ Processing [%d/%d]: %s\n", i+1, len(markdownFiles), filepath.Base(filePath))

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
			fmt.Printf("▪︎ Error extracting from file %s: %v\n", filePath, err)
			continue
		}

		// Display results with improved formatting
		fmt.Printf("▪︎ Organisation: %s (Type: %s)\n", result.Organisation.Name, result.Organisation.DocumentType)
		fmt.Printf("▪︎ Financials: Revenue $%.2f, Budget $%.2f\n", result.Organisation.Revenue, result.Organisation.Budget)
		fmt.Printf("▪︎ Contact: %s (%s)\n", result.Organisation.Contact.Name, result.Organisation.Contact.Email)
		fmt.Printf("▪︎ Projects: %d found\n", len(result.Organisation.Projects))
		for j, proj := range result.Organisation.Projects {
			fmt.Printf("   %d. %s (%s) - $%.2f\n", j+1, proj.Name, proj.Status, proj.Budget)
		}
	}
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
