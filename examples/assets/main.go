package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// DocumentMetadata represents basic document information
type DocumentMetadata struct {
	Title       string `json:"title" unstruct:"basic"`
	Description string `json:"description" unstruct:"basic"`
	Category    string `json:"category" unstruct:"basic"`
	Author      string `json:"author" unstruct:"person"`
	Date        string `json:"date" unstruct:"basic"`
	Version     string `json:"version" unstruct:"basic"`
}

// ProjectInfo represents project-specific information extracted from documents
type ProjectInfo struct {
	ProjectCode string  `json:"projectCode" unstruct:"project"`
	ProjectName string  `json:"projectName" unstruct:"project"`
	Budget      float64 `json:"budget" unstruct:"project"`
	Currency    string  `json:"currency" unstruct:"project"`
	StartDate   string  `json:"startDate" unstruct:"project"`
	EndDate     string  `json:"endDate" unstruct:"project"`
	Status      string  `json:"status" unstruct:"project"`
	Priority    string  `json:"priority" unstruct:"project"`
	ProjectLead string  `json:"projectLead" unstruct:"person"`
	TeamSize    int     `json:"teamSize" unstruct:"project"`
}

func main() {
	ctx := context.Background()

	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	// Create Google GenAI client
	fmt.Println("Creating Google GenAI client...")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Create Stick-based prompt provider from template files
	fmt.Println("Setting up Stick template engine...")
	promptProvider, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
	)
	if err != nil {
		log.Fatal("Failed to create Stick prompt provider:", err)
	}

	// Example 1: Text-only document extraction
	fmt.Println("\n=== Text Document Example ===")
	runTextExample(ctx, client, promptProvider)

	// Example 2: File upload and extraction from markdown files
	fmt.Println("\n=== File Upload Examples ===")
	runFileUploadExamples(ctx, client, promptProvider)

	// Example 3: Dry run for cost estimation
	fmt.Println("\n=== Dry Run Example ===")
	runDryRunExample(ctx, client, promptProvider)
}

func runTextExample(ctx context.Context, client *genai.Client, promptProvider unstruct.PromptProvider) {
	// Create unstructor for basic document metadata
	u := unstruct.New[DocumentMetadata](client, promptProvider)

	textDoc := `Technical Report: Advanced AI Systems
	
	This document describes the implementation of machine learning algorithms for natural language processing.
	The report was authored by Dr. Sarah Johnson on January 15, 2024.
	Document version: 1.2
	Category: Technology Research`

	textAsset := unstruct.NewTextAsset(textDoc)
	assets := []unstruct.Asset{textAsset}

	result, err := u.Unstruct(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error extracting from text: %v", err)
		return
	}

	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Description: %s\n", result.Description)
	fmt.Printf("Category: %s\n", result.Category)
	fmt.Printf("Author: %s\n", result.Author)
	fmt.Printf("Date: %s\n", result.Date)
	fmt.Printf("Version: %s\n", result.Version)
}

func runFileUploadExamples(ctx context.Context, client *genai.Client, promptProvider unstruct.PromptProvider) {
	// Find markdown files in the docs directory
	markdownFiles := findMarkdownFiles("docs")
	if len(markdownFiles) == 0 {
		fmt.Println("No markdown files found in docs/ directory")
		return
	}

	// Create unstructor for project information
	u := unstruct.New[ProjectInfo](client, promptProvider)

	for _, filePath := range markdownFiles {
		fmt.Printf("\n--- Processing: %s ---\n", filepath.Base(filePath))

		// Create FileAsset that will upload the file to Files API
		fileAsset := unstruct.NewFileAsset(
			client,
			filePath,
			unstruct.WithDisplayName(fmt.Sprintf("Document Analysis - %s", filepath.Base(filePath))),
		)

		assets := []unstruct.Asset{fileAsset}

		result, err := u.Unstruct(
			ctx,
			assets,
			unstruct.WithModel("gemini-1.5-pro"), // Use Pro model for file analysis
		)
		if err != nil {
			log.Printf("Error extracting from file %s: %v", filePath, err)
			continue
		}

		// Display results
		fmt.Printf("Project Code: %s\n", result.ProjectCode)
		fmt.Printf("Project Name: %s\n", result.ProjectName)
		fmt.Printf("Budget: %.2f %s\n", result.Budget, result.Currency)
		fmt.Printf("Timeline: %s to %s\n", result.StartDate, result.EndDate)
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Priority: %s\n", result.Priority)
		fmt.Printf("Project Lead: %s\n", result.ProjectLead)
		fmt.Printf("Team Size: %d\n", result.TeamSize)
	}
}

func runDryRunExample(ctx context.Context, client *genai.Client, promptProvider unstruct.PromptProvider) {
	// Create unstructor for basic document metadata
	u := unstruct.New[DocumentMetadata](client, promptProvider)

	textDoc := "Sample document for cost estimation"
	textAsset := unstruct.NewTextAsset(textDoc)
	assets := []unstruct.Asset{textAsset}

	stats, err := u.DryRun(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error in dry run: %v", err)
		return
	}

	fmt.Printf("Estimated prompt calls: %d\n", stats.PromptCalls)
	fmt.Printf("Estimated input tokens: %d\n", stats.TotalInputTokens)
	fmt.Printf("Estimated output tokens: %d\n", stats.TotalOutputTokens)
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
