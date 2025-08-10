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

// Example destination struct for extracting customer information
type CustomerInfo struct {
	Name     string  `json:"name" unstruct:"customer"`
	Email    string  `json:"email" unstruct:"customer"`
	Phone    string  `json:"phone" unstruct:"customer"`
	Company  string  `json:"company" unstruct:"business"`
	Industry string  `json:"industry" unstruct:"business"`
	Revenue  float64 `json:"revenue" unstruct:"financial"`
	Budget   float64 `json:"budget" unstruct:"financial"`
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
	slog.Debug("Starting OpenAI example", "context", "background")

	// Check for required environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_KEY") // Fallback to OPENAI_KEY
	}
	if apiKey == "" {
		slog.Debug("Neither OPENAI_API_KEY nor OPENAI_KEY found in environment")
		fmt.Println("Error: OPENAI_API_KEY or OPENAI_KEY environment variable is required")
		fmt.Println("Please set it with your OpenAI API key:")
		fmt.Println("export OPENAI_API_KEY=your_api_key_here")
		fmt.Println("or")
		fmt.Println("export OPENAI_KEY=your_api_key_here")
		os.Exit(1)
	}
	slog.Debug("Found OpenAI API key", "key_length", len(apiKey))

	// Note: This example demonstrates the pattern for OpenAI integration
	// In production, you would use proper OpenAI client configuration
	fmt.Println("Configuring for OpenAI-style usage...")
	slog.Debug("Setting up OpenAI-style configuration")

	// Check for required environment variable
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		slog.Debug("GEMINI_API_KEY not found in environment")
		fmt.Println("Error: GEMINI_API_KEY environment variable is required")
		fmt.Println("Please set it with your Google AI API key:")
		fmt.Println("export GEMINI_API_KEY=your_api_key_here")
		fmt.Println("Note: This example uses Gemini as fallback for demonstration")
		os.Exit(1)
	}
	slog.Debug("Found GEMINI_API_KEY", "key_length", len(geminiKey))
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
		APIKey:  geminiKey,
	})
	if err != nil {
		slog.Debug("Google AI client creation failed", "error", err)
		fmt.Printf("Failed to create Google AI client: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Google AI client created successfully")

	// 1. Prompts for each extractor tag ⭐️
	// 1️⃣
	slog.Debug("Setting up prompts", "prompt_count", 3)
	prompts := filePrompts{
		"customer":  `Extract customer personal information from this text and return as JSON with fields: {{.Keys}}. Return only valid JSON.`,
		"business":  `Extract business information from this text and return as JSON with fields: {{.Keys}}. Return only valid JSON.`,
		"financial": `Extract financial information from this text and return as JSON with fields: {{.Keys}}. Return numeric values for revenue and budget as numbers, not strings. Example: {"revenue": 1500000, "budget": 250000}.`,
	}
	slog.Debug("prompts configured", "customer_prompt_length", len(prompts["customer"]), "business_prompt_length", len(prompts["business"]), "financial_prompt_length", len(prompts["financial"]))

	// 2. Build unstructor
	// 2️⃣
	slog.Debug("Creating Unstructor")
	uno := unstruct.New[CustomerInfo](client, prompts)
	slog.Debug("Unstructor created successfully")

	// 3. Run extraction
	// 3️⃣
	doc := `John Smith (john.smith@techcorp.com, +1-555-0123) is the CTO at TechCorp, a software development company in the technology industry. The company has annual revenue of $1.5M and they have a budget of $250K for new AI initiatives this year.`
	fmt.Printf("Extracting customer information from document: %s\n", doc)
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
	slog.Debug("Extraction completed successfully",
		"name", out.Name,
		"email", out.Email,
		"phone", out.Phone,
		"company", out.Company,
		"industry", out.Industry,
		"revenue", out.Revenue,
		"budget", out.Budget)

	fmt.Printf("Extraction Results:\n")
	fmt.Printf("Name: %s\n", out.Name)
	fmt.Printf("Email: %s\n", out.Email)
	fmt.Printf("Phone: %s\n", out.Phone)
	fmt.Printf("Company: %s\n", out.Company)
	fmt.Printf("Industry: %s\n", out.Industry)
	fmt.Printf("Revenue: $%.2f\n", out.Revenue)
	fmt.Printf("Budget: $%.2f\n", out.Budget)
	fmt.Printf("Full result: %+v\n", *out)
	slog.Debug("OpenAI example completed successfully")
}
