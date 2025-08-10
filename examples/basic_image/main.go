package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

type ImageExtractionRequest struct {
	Document struct {
		CompanyName  string  `json:"company_name"`
		DocumentType string  `json:"document_type"`
		TotalAmount  float64 `json:"total_amount"`
	} `unstruct:"prompt/basic/model/gemini-1.5-pro?temperature=0"`
}

func main() {
	ctx := context.Background()

	fmt.Println("Starting image extraction application...")

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: GEMINI_API_KEY environment variable is required")
		os.Exit(1)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	prompts, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "prompts"),
	)
	if err != nil {
		log.Fatal("Failed to create prompt provider:", err)
	}

	// Set log level to INFO to reduce noise
	slog.SetLogLoggerLevel(slog.LevelInfo)
	extractor := unstruct.NewWithLogger[ImageExtractionRequest](client, prompts, slog.Default())

	// Create image asset from file
	imageData, err := os.ReadFile("docs/image.png")
	if err != nil {
		log.Fatal("Failed to read image file:", err)
	}

	imageAsset := unstruct.NewImageAsset(imageData, "image/png")

	fmt.Println("=== IMAGE DOCUMENT EXTRACTION ===")
	fmt.Printf("Processing image: docs/image.png\n")

	result, err := extractor.Unstruct(
		ctx,
		[]unstruct.Asset{imageAsset},
	)
	if err != nil {
		log.Fatal("Extraction failed:", err)
	}

	fmt.Printf("Results:\n")
	fmt.Printf("  Company Name: '%s'\n", result.Document.CompanyName)
	fmt.Printf("  Document Type: '%s'\n", result.Document.DocumentType)
	fmt.Printf("  Total Amount: %.2f\n", result.Document.TotalAmount)

	// Validation checks
	if result.Document.CompanyName != "TechCorp Inc." {
		panic(fmt.Sprintf("Expected company name 'TechCorp Inc.' but got '%s'", result.Document.CompanyName))
	}
	if result.Document.DocumentType != "Annual Report" {
		panic(fmt.Sprintf("Expected document type 'Annual Report' but got '%s'", result.Document.DocumentType))
	}
	if result.Document.TotalAmount != 1000.0 {
		panic(fmt.Sprintf("Expected total amount 1000.0 but got %.2f", result.Document.TotalAmount))
	}
}
