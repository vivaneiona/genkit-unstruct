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

type Document struct {
	CompanyName  string  `json:"company_name"`
	DocumentType string  `json:"document_type"`
	TotalAmount  float64 `json:"total_amount"`
}
type DocumentsExtractionRequest struct {
	Documents []Document `unstruct:"prompt/basic/model/gemini-1.5-pro?temperature=0"`
}

func main() {
	ctx := context.Background()

	fmt.Println("Starting multi-file extraction application...")

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
	extractor := unstruct.NewWithLogger[DocumentsExtractionRequest](client, prompts, slog.Default())

	// Create file assets for the two documents
	fooAsset := unstruct.NewFileAsset(client, "docs/foo.md",
		unstruct.WithMimeType("text/markdown"),
		unstruct.WithDisplayName("Foo Company Report"),
	)

	booAsset := unstruct.NewFileAsset(client, "docs/boo.md",
		unstruct.WithMimeType("text/markdown"),
		unstruct.WithDisplayName("Boo Company Report"),
	)

	fmt.Println("=== MULTI-FILE DOCUMENT EXTRACTION ===")
	fmt.Printf("Processing files: docs/foo.md, docs/boo.md\n")

	result, err := extractor.Unstruct(
		ctx,
		[]unstruct.Asset{fooAsset, booAsset},
	)
	if err != nil {
		log.Fatal("Extraction failed:", err)
	}

	fmt.Printf("Results:\n")
	fmt.Printf("Found %d documents:\n", len(result.Documents))

	for i, doc := range result.Documents {
		fmt.Printf("  Document %d:\n", i+1)
		fmt.Printf("    Company Name: '%s'\n", doc.CompanyName)
		fmt.Printf("    Document Type: '%s'\n", doc.DocumentType)
		fmt.Printf("    Total Amount: %.2f\n", doc.TotalAmount)
	}

	// Validation checks - expect 2 documents
	if len(result.Documents) != 2 {
		panic(fmt.Sprintf("Expected 2 documents but got %d", len(result.Documents)))
	}

	// Check first document (foo.md)
	doc1 := result.Documents[0]
	if doc1.CompanyName != "Foo." {
		panic(fmt.Sprintf("Expected first document company name 'Foo.' but got '%s'", doc1.CompanyName))
	}
	if doc1.DocumentType != "Annual Report" {
		panic(fmt.Sprintf("Expected first document type 'Annual Report' but got '%s'", doc1.DocumentType))
	}
	if doc1.TotalAmount != 1000.0 {
		panic(fmt.Sprintf("Expected first document total amount 1000.0 but got %.2f", doc1.TotalAmount))
	}

	// Check second document (boo.md)
	doc2 := result.Documents[1]
	if doc2.CompanyName != "Boo" {
		panic(fmt.Sprintf("Expected second document company name 'Boo' but got '%s'", doc2.CompanyName))
	}
	if doc2.DocumentType != "Report" {
		panic(fmt.Sprintf("Expected second document type 'Report' but got '%s'", doc2.DocumentType))
	}
	if doc2.TotalAmount != 1500.0 {
		panic(fmt.Sprintf("Expected second document total amount 1500.0 but got %.2f", doc2.TotalAmount))
	}

	fmt.Println("All validations passed!")
}
