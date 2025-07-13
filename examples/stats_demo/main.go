package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

type TestDocument struct {
	Name    string `json:"name" unstruct:"person-info,gpt-4o"`
	Age     int    `json:"age" unstruct:"person-info,gpt-4o"`
	Email   string `json:"email" unstruct:"contact-info,gpt-3.5-turbo"`
	Address string `json:"address" unstruct:"contact-info,gpt-3.5-turbo"`
}

func main() {
	fmt.Println("=== Execution Statistics Collection Demo ===")

	// Sample document for testing
	sampleDoc := `
		John Doe is 35 years old. His email is john.doe@example.com.
		He lives at 123 Main Street, Springfield, IL 62701.
	`

	// Initialize Genkit client
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Note: GEMINI_API_KEY not set, using demo mode")
		demoWithoutAPI(sampleDoc)
		return
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	// Create Unstructor with logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create a simple prompt provider
	promptProvider, err := unstruct.NewStickPromptProvider()
	if err != nil {
		fmt.Printf("Failed to create prompt provider: %v\n", err)
		return
	}

	unstructor := unstruct.NewWithLogger[TestDocument](client, promptProvider, logger)

	// 1. Test Dry-Run Statistics Collection
	fmt.Println("\n1. Testing Dry-Run Statistics Collection")
	assets := []unstruct.Asset{unstruct.NewTextAsset(sampleDoc)}
	stats, err := unstructor.DryRun(context.Background(), assets, unstruct.WithModel("gpt-3.5-turbo"))
	if err != nil {
		fmt.Printf("Dry-run failed: %v\n", err)
		return
	}

	fmt.Printf("Prompt Calls: %d\n", stats.PromptCalls)
	fmt.Printf("Prompt Groups: %d\n", stats.PromptGroups)
	fmt.Printf("Fields Extracted: %d\n", stats.FieldsExtracted)
	fmt.Printf("Total Input Tokens: %d\n", stats.TotalInputTokens)
	fmt.Printf("Total Output Tokens: %d\n", stats.TotalOutputTokens)
	fmt.Printf("Model Calls: %v\n", stats.ModelCalls)

	fmt.Println("\nGroup Details:")
	for i, group := range stats.GroupDetails {
		fmt.Printf("  Group %d:\n", i+1)
		fmt.Printf("    Prompt: %s\n", group.PromptName)
		fmt.Printf("    Model: %s\n", group.Model)
		fmt.Printf("    Fields: %v\n", group.Fields)
		fmt.Printf("    Input Tokens: %d\n", group.InputTokens)
		fmt.Printf("    Output Tokens: %d\n", group.OutputTokens)
	}

	// 2. Test Enhanced Plan Building with Dry-Run
	fmt.Println("\n2. Testing Enhanced Plan Building with Dry-Run")

	// Configure models for different fields
	modelConfig := map[string]string{
		"name":    "gpt-4o",
		"age":     "gpt-4o",
		"email":   "gpt-3.5-turbo",
		"address": "gpt-3.5-turbo",
	}

	// Build plan using dry-run execution
	builder := unstruct.NewPlanBuilder().
		WithSchema(map[string]interface{}{"fields": []string{"name", "age", "email", "address"}}).
		WithModelConfig(modelConfig).
		WithUnstructor(unstructor).
		WithSampleDocument(sampleDoc)

	plan, err := builder.Explain()
	if err != nil {
		fmt.Printf("Failed to build plan: %v\n", err)
		return
	}

	// Print the enhanced plan
	fmt.Printf("Expected Models: %v\n", plan.ExpectedModels)
	fmt.Printf("Expected Call Counts: %v\n", plan.ExpectedCallCounts)

	// 3. Show JSON output
	fmt.Println("\n3. Plan JSON Output:")
	planJSON, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling plan: %v\n", err)
		return
	}
	fmt.Println(string(planJSON))

	// 4. Compare Expected vs Actual (simulation)
	fmt.Println("\n4. Expected vs Actual Comparison:")
	fmt.Printf("Expected prompt calls: %d\n", len(plan.ExpectedCallCounts))
	fmt.Printf("Actual prompt calls: %d\n", stats.PromptCalls)

	for model, expectedCalls := range plan.ExpectedCallCounts {
		actualCalls := stats.ModelCalls[model]
		status := "✓"
		if expectedCalls != actualCalls {
			status = "⚠"
		}
		fmt.Printf("%s Model %s: expected %d, actual %d\n", status, model, expectedCalls, actualCalls)
	}
}

func demoWithoutAPI(sampleDoc string) {
	fmt.Println("Running in demo mode without API...")

	// Simulate statistics for demo
	stats := &unstruct.ExecutionStats{
		PromptCalls:       2,
		ModelCalls:        map[string]int{"gpt-4o": 1, "gpt-3.5-turbo": 1},
		PromptGroups:      2,
		FieldsExtracted:   4,
		TotalInputTokens:  250,
		TotalOutputTokens: 80,
		GroupDetails: []unstruct.GroupExecution{
			{
				PromptName:   "person-info",
				Model:        "gpt-4o",
				Fields:       []string{"name", "age"},
				InputTokens:  120,
				OutputTokens: 40,
			},
			{
				PromptName:   "contact-info",
				Model:        "gpt-3.5-turbo",
				Fields:       []string{"email", "address"},
				InputTokens:  130,
				OutputTokens: 40,
			},
		},
	}

	fmt.Printf("Demo Statistics:\n")
	fmt.Printf("Prompt Calls: %d\n", stats.PromptCalls)
	fmt.Printf("Model Calls: %v\n", stats.ModelCalls)
	fmt.Printf("Total Tokens: %d input + %d output\n", stats.TotalInputTokens, stats.TotalOutputTokens)
}
