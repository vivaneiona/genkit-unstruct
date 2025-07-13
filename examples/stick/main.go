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

// Project represents the extracted data structure
type Project struct {
	ProjectCode string  `json:"projectCode" unstruct:"code"`
	CertIssuer  string  `json:"certIssuer"  unstruct:"cert"`
	Latitude    float64 `json:"lat" unstruct:"default"` // explicit prompt
	Longitude   float64 `json:"lon" unstruct:"default"` // explicit prompt
}

func main() {
	// Set up colored logging with tint
	logger := slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:   slog.LevelDebug,
			NoColor: false,
		}),
	)
	slog.SetDefault(logger)

	ctx := context.Background()
	slog.Debug("Starting Stick template example")

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

	// Create Google AI client for unstract
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

	// Create Stick-based prompt provider from template files
	slog.Debug("Creating Stick prompt provider", "template_path", "templates")
	promptProvider, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
	)
	if err != nil {
		slog.Debug("Stick prompt provider creation failed", "error", err)
		fmt.Printf("Failed to create Stick prompt provider: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Stick prompt provider created successfully")

	// Build extractor with Stick templates
	slog.Debug("Creating Unstract with Stick templates")
	uno := unstruct.New[Project](client, promptProvider)

	// Test documents with different complexity
	testDocuments := []struct {
		name string
		text string
	}{
		{
			name: "Simple Station Document",
			text: `Station 512-B. Certificate by "MegaTel". Coords: 13.75, 100.52.`,
		},
		{
			name: "Complex Project Document",
			text: `PROJECT ALPHA-7 Status Report
			Location: Research facility at coordinates 40.7128°N, 74.0060°W
			Security Certificate issued by Quantum Systems Corp
			All systems operational as of 2024-01-15.`,
		},
		{
			name: "Multiple Identifiers Document",
			text: `Facility Code: XR-9001, Backup ID: SITE-42
			Certificate Authority: "Advanced Technologies LLC"
			GPS Location: 35.6762, 139.6503 (Tokyo Bay Area)`,
		},
	}

	fmt.Println("🎯 Stick Template Engine Extraction Demo")
	fmt.Println("=========================================")

	for i, doc := range testDocuments {
		fmt.Printf("\n📄 Document %d: %s\n", i+1, doc.name)
		fmt.Printf("Text: %s\n", doc.text)
		fmt.Println("---")

		slog.Debug("Starting extraction", "document", doc.name, "model", "gemini-1.5-flash")

		assets := []unstruct.Asset{unstruct.NewTextAsset(doc.text)}
		out, err := uno.Unstruct(
			context.Background(),
			assets,
			unstruct.WithModel("gemini-1.5-flash"),
			unstruct.WithTimeout(30*time.Second),
			unstruct.WithRetry(2, 1*time.Second),
		)
		if err != nil {
			slog.Debug("Extraction failed", "document", doc.name, "error", err)
			fmt.Printf("❌ Failed to extract information: %v\n", err)
			continue
		}

		slog.Debug("Extraction completed successfully",
			"document", doc.name,
			"project_code", out.ProjectCode,
			"cert_issuer", out.CertIssuer,
			"latitude", out.Latitude,
			"longitude", out.Longitude)

		fmt.Printf("✅ Extraction Results:\n")
		fmt.Printf("   📊 Project Code: %s\n", out.ProjectCode)
		fmt.Printf("   📜 Certificate Issuer: %s\n", out.CertIssuer)
		fmt.Printf("   📍 Coordinates: %.6f, %.6f\n", out.Latitude, out.Longitude)
		fmt.Printf("   🔍 Full result: %+v\n", *out)
	}

	fmt.Println("\n🎉 Stick template extraction demo completed!")
	slog.Debug("Example completed successfully")
}
