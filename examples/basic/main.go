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

type SimpleExtractionRequest struct {
	Company struct {
		CompanyName    string  `json:"company_name"`
		Capitalization float64 `json:"capitalization"`
		DocumentType   string  `json:"document_type"`
	} `unstruct:"prompt/basic/model/gemini-1.5-flash?temperature=0"`
}

func main() {
	ctx := context.Background()

	fmt.Println("Starting application...")

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
	slog.SetLogLoggerLevel(slog.LevelDebug)
	extractor := unstruct.NewWithLogger[SimpleExtractionRequest](client, prompts, slog.Default())

	testDoc := "Company: TechCorp Inc. Capitalization: 1000$. Type: Annual Report"
	fmt.Printf("Input: %s\n", testDoc)

	result, err := extractor.Unstruct(
		ctx,
		unstruct.AssetsFrom(testDoc),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Results:\n")
	fmt.Printf("  Company Name: '%s'\n", result.Company.CompanyName)
	fmt.Printf("  Capitalization: %.2f\n", result.Company.Capitalization)
	fmt.Printf("  Document Type: '%s'\n", result.Company.DocumentType)
	if result.Company.CompanyName != "TechCorp Inc." {
		panic(result.Company.CompanyName)
	}
	if result.Company.Capitalization != 1000.0 {
		panic(result.Company.Capitalization)
	}

}
