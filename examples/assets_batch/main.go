package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// ProjectInfo represents project information extracted from documents
type ProjectInfo struct {
	ProjectCode string `json:"projectCode" unstruct:"prompt/project/model/gemini-1.5-flash"`
	ProjectName string `json:"projectName" unstruct:"prompt/project/model/gemini-1.5-flash"`
	Budget      string `json:"budget" unstruct:"prompt/financial/model/gemini-1.5-pro?temperature=0.1&topK=20"`
	Currency    string `json:"currency" unstruct:"prompt/financial/model/gemini-1.5-pro?temperature=0.1&topK=20"`
	StartDate   string `json:"startDate" unstruct:"prompt/timeline/model/gemini-1.5-flash"`
	EndDate     string `json:"endDate" unstruct:"prompt/timeline/model/gemini-1.5-flash"`
	Status      string `json:"status" unstruct:"prompt/project/model/gemini-1.5-flash"`
	Priority    string `json:"priority" unstruct:"prompt/project/model/gemini-1.5-flash"`
	ProjectLead string `json:"projectLead" unstruct:"prompt/person/model/gemini-1.5-pro?temperature=0.2&topK=40"`
	TeamSize    int    `json:"teamSize" unstruct:"prompt/project/model/gemini-1.5-flash"`
}

func main() {
	ctx := context.Background()

	// Enable debug logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	// Create Google GenAI client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Create prompt provider with improved prompts for different field types
	prompts := unstruct.SimplePromptProvider{
		"project":   "Extract project information from the document. Return JSON with fields: {{.Keys}}. For example: {\"projectCode\": \"PROJ-123\", \"projectName\": \"Example Project\", \"status\": \"Active\", \"priority\": \"High\", \"teamSize\": 5}. Document: {{.Document}}",
		"financial": "Extract financial data from the document. Return JSON with budget as a string number and currency. For example: {\"budget\": \"150000.00\", \"currency\": \"USD\"}. Fields: {{.Keys}}. Document: {{.Document}}",
		"timeline":  "Extract timeline/date information from the document. Return JSON with date strings. For example: {\"startDate\": \"January 1, 2024\", \"endDate\": \"December 31, 2024\"}. Fields: {{.Keys}}. Document: {{.Document}}",
		"person":    "Extract person/contact information from the document. Return JSON with name strings. For example: {\"projectLead\": \"John Smith\"}. Fields: {{.Keys}}. Document: {{.Document}}",
	}

	// Run batch processing example
	runBatchExample(ctx, client, prompts, logger)
}

func runBatchExample(ctx context.Context, client *genai.Client, prompts unstruct.PromptProvider, logger *slog.Logger) {
	// Find markdown files to process
	files := findMarkdownFiles("docs")
	if len(files) == 0 {
		log.Fatal("No markdown files found in docs/ directory")
	}

	fmt.Printf("Found %d files to process\n", len(files))

	// Create batch asset and unstructor
	batchAsset := unstruct.NewBatchFileAsset(client, files)
	u := unstruct.NewWithLogger[ProjectInfo](client, prompts, logger)
	assets := []unstruct.Asset{batchAsset}

	// Show execution plan
	fmt.Println("\n=== Execution Plan ===")
	plan, err := u.Explain(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error generating plan: %v", err)
		return
	}
	fmt.Println(plan)

	// Get cost estimation
	stats, err := u.DryRun(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error in cost estimation: %v", err)
		return
	}

	// Display cost estimation
	inputCost := float64(stats.TotalInputTokens) / 1000000 * 0.075 // Gemini Flash pricing
	outputCost := float64(stats.TotalOutputTokens) / 1000000 * 0.30
	fmt.Printf("\nEstimated cost: $%.6f (%d calls, %d input tokens, %d output tokens)\n",
		inputCost+outputCost, stats.PromptCalls, stats.TotalInputTokens, stats.TotalOutputTokens)

	// Process files
	fmt.Println("\n=== Processing Files ===")
	startTime := time.Now()
	result, err := u.Unstruct(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error processing: %v", err)
		return
	}
	duration := time.Since(startTime)

	// Debug: Print raw result
	fmt.Printf("\n=== Debug: Raw Result ===\n")
	fmt.Printf("ProjectCode: '%s'\n", result.ProjectCode)
	fmt.Printf("ProjectName: '%s'\n", result.ProjectName)
	fmt.Printf("Budget: %f\n", result.Budget)
	fmt.Printf("Currency: '%s'\n", result.Currency)
	fmt.Printf("StartDate: '%s'\n", result.StartDate)
	fmt.Printf("EndDate: '%s'\n", result.EndDate)
	fmt.Printf("Status: '%s'\n", result.Status)
	fmt.Printf("Priority: '%s'\n", result.Priority)
	fmt.Printf("ProjectLead: '%s'\n", result.ProjectLead)
	fmt.Printf("TeamSize: %d\n", result.TeamSize)

	// Display results
	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Project: %s (%s)\n", result.ProjectName, result.ProjectCode)
	fmt.Printf("Budget: %.2f %s\n", result.Budget, result.Currency)
	fmt.Printf("Timeline: %s to %s\n", result.StartDate, result.EndDate)
	fmt.Printf("Status: %s (Priority: %s)\n", result.Status, result.Priority)
	fmt.Printf("Lead: %s (Team: %d)\n", result.ProjectLead, result.TeamSize)
	fmt.Printf("Processing time: %v\n", duration)

	// Cleanup
	if err := batchAsset.Cleanup(ctx); err != nil {
		log.Printf("Cleanup warning: %v", err)
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
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files
}
