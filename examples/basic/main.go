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
	"github.com/vivaneiona/genkit-unstruct"
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

// Example destination struct
type Project struct {
	ProjectCode string  `json:"projectCode" unstruct:"code"`
	CertIssuer  string  `json:"certIssuer"  unstruct:"cert"`
	Latitude    float64 `json:"lat"` // default extractor tag
	Longitude   float64 `json:"lon"`
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
		"code":      `Extract the project code from this text and return as JSON with fields: {{.Keys}}. Return only valid JSON. Text: {{.Document}}`,
		"cert":      `Find the certificate issuer from this text and return as JSON with fields: {{.Keys}}. Return only valid JSON. Text: {{.Document}}`,
		"default":   `Extract location coordinates from this text and return as JSON with fields: {{.Keys}}. Return numeric values for coordinates as numbers, not strings. Example: {"lat": 13.75, "lon": 100.52}. Text: {{.Document}}`,
		"extractor": `Extract location coordinates from this text and return as JSON with fields: {{.Keys}}. Return numeric values for coordinates as numbers, not strings. Example: {"lat": 13.75, "lon": 100.52}. Text: {{.Document}}`,
	}
	slog.Debug("prompts configured", "code_prompt_length", len(prompts["code"]), "cert_prompt_length", len(prompts["cert"]), "default_prompt_length", len(prompts["default"]), "extractor_prompt_length", len(prompts["extractor"]))

	// 2. Build unstructor
	// 2️⃣
	slog.Debug("Creating Unstructor")
	uno := unstruct.New[Project](client, prompts)
	slog.Debug("Unstructor created successfully")

	// 3. Run extraction
	// 3️⃣
	doc := `Station 512-B. Certificate by "MegaTel". Coords: 13.75, 100.52.`
	fmt.Printf("Extracting information from document: %s\n", doc)
	slog.Debug("Starting extraction", "document", doc, "model", "gemini-1.5-flash", "timeout", "30s")

	out, err := uno.Unstruct(
		context.Background(),
		doc,
		unstruct.WithModel("gemini-1.5-flash"),
		unstruct.WithTimeout(30*time.Second),
	)
	if err != nil {
		slog.Debug("Extraction failed", "error", err)
		fmt.Printf("Failed to extract information: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Extraction completed successfully", "project_code", out.ProjectCode, "cert_issuer", out.CertIssuer, "latitude", out.Latitude, "longitude", out.Longitude)

	fmt.Printf("Extraction Results:\n")
	fmt.Printf("Project Code: %s\n", out.ProjectCode)
	fmt.Printf("Certificate Issuer: %s\n", out.CertIssuer)
	fmt.Printf("Latitude: %f\n", out.Latitude)
	fmt.Printf("Longitude: %f\n", out.Longitude)
	fmt.Printf("Full result: %+v\n", *out)
	slog.Debug("Example completed successfully")
}
