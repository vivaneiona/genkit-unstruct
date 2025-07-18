package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/lmittmann/tint"
	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// Fake prompt store
type filePrompts map[string]string

func (p filePrompts) GetPrompt(tag string, _ int) (string, error) {
	if s, ok := p[tag]; ok {
		slog.Debug("Found prompt", "tag", tag, "prompt_length", len(s))
		return s, nil
	}
	slog.Debug("Prompt not found", "tag", tag, "available_tags", getKeys(p))
	return "", fmt.Errorf("prompt %q not found", tag)
}

func getKeys(m filePrompts) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Business document structure with model selection per field type
type ExtractionRequest struct {
	Organisation struct {
		// Basic information - uses fast model
		Name         string `json:"name"`    // inherited unstruct:"prompt/basic/model/gemini-1.5-flash"
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
	// Set up colored logging with tint to see debug messages
	logger := slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:   slog.LevelDebug,
			NoColor: false,
		}),
	)
	slog.SetDefault(logger)

	ctx := context.Background()
	slog.Debug("Starting example", "context", "background")

	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		slog.Debug("GEMINI_API_KEY not found in environment")
		fmt.Println("Error: GEMINI_API_KEY environment variable is required")
		fmt.Println("Please set it with your Google AI API key:")
		fmt.Println("export GEMINI_API_KEY=your_api_key_here")
		os.Exit(1)
	}
	slog.Debug("Found GEMINI_API_KEY", "key_length", len(apiKey))

	// Initialize Genkit with GoogleAI plugin
	fmt.Println("Initializing Genkit with GoogleAI plugin...")
	slog.Debug("Initializing Genkit with GoogleAI plugin")
	_, err := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))
	if err != nil {
		slog.Debug("Genkit initialization failed", "error", err)
		fmt.Printf("Failed to initialize Genkit: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Genkit initialization completed successfully")

	// Create Google AI client for unstruct
	slog.Debug("Creating Google AI client", "backend", "GeminiAPI")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		slog.Debug("Google AI client creation failed", "error", err)
		fmt.Printf("Failed to create Google AI client: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Google AI client created successfully")

	// 1. Prompts for each extractor tag ⭐️
	// 1️⃣
	slog.Debug("Setting up prompts", "prompt_count", 4)
	prompts := filePrompts{
		"basic":     "Extract basic info: {{.Keys}} from: {{.Document}}. Return JSON with exact field structure.",
		"financial": "Find financial data ({{.Keys}}) in: {{.Document}}. Return numeric values only (e.g., 2500000 for $2.5M). Use exact JSON structure: {\"organisation\": {\"revenue\": number, \"budget\": number}}",
		"contact":   "Extract contact details ({{.Keys}}) from: {{.Document}}. Return JSON with exact field structure.",
		"projects":  "List all projects with {{.Keys}} from: {{.Document}}. Return budget as numeric values only (e.g., 500000 for $500K). Use exact JSON structure: {\"organisation\": {\"projects\": [{\"name\": string, \"status\": string, \"budget\": number}]}}",
	}
	slog.Debug("prompts configured", "basic_prompt_length", len(prompts["basic"]), "financial_prompt_length", len(prompts["financial"]), "contact_prompt_length", len(prompts["contact"]), "projects_prompt_length", len(prompts["projects"]))

	// 2. Build unstructor
	// 2️⃣
	slog.Debug("Creating Unstructor")
	uno := unstruct.New[ExtractionRequest](client, prompts)
	slog.Debug("Unstructor created successfully")

	// 3. Run extraction
	// 3️⃣
	doc := `TechCorp Inc. Annual Report 2024. Revenue: $2.5M, Budget: $3.0M. Contact: John Smith (john@techcorp.com, +1-555-0123). Projects: Alpha (Active, $500K), Beta (Planning, $800K).`
	fmt.Printf("Extracting information from document: %s\n", doc)
	slog.Debug("Starting extraction", "document", doc, "model", "gemini-1.5-flash", "timeout", "30s")

	out, err := uno.Unstruct(
		context.Background(),
		unstruct.AssetsFrom(doc),
		unstruct.WithModel("gemini-1.5-flash"),
		unstruct.WithTimeout(30*time.Second),
	)
	if err != nil {
		slog.Debug("Extraction failed", "error", err)
		fmt.Printf("Failed to extract information: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Extraction completed successfully", "organisation_name", out.Organisation.Name, "document_type", out.Organisation.DocumentType, "revenue", out.Organisation.Revenue, "budget", out.Organisation.Budget)

	fmt.Printf("Extraction Results:\n")
	fmt.Printf("Organisation: %s (Type: %s)\n", out.Organisation.Name, out.Organisation.DocumentType)
	fmt.Printf("Financials: Revenue $%.2f, Budget $%.2f\n", out.Organisation.Revenue, out.Organisation.Budget)
	fmt.Printf("Contact: %s (%s)\n", out.Organisation.Contact.Name, out.Organisation.Contact.Email)
	fmt.Printf("Projects: %d found\n", len(out.Organisation.Projects))
	fmt.Printf("Full result: %+v\n", *out)
	slog.Debug("Example completed successfully")
}
