package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

type TestDocument struct {
	Name    string `json:"name" unstruct:"person-info,gpt-4o"`
	Age     int    `json:"age" unstruct:"person-info,gpt-4o"`
	Email   string `json:"email" unstruct:"contact-info,gpt-3.5-turbo"`
	Address string `json:"address" unstruct:"contact-info,gpt-3.5-turbo"`
}

// SimplePromptProvider provides basic prompt templates for demo
type SimplePromptProvider struct {
	templates map[string]string
}

func NewSimplePromptProvider() *SimplePromptProvider {
	return &SimplePromptProvider{
		templates: map[string]string{
			"person-info": `Extract personal information from the following document:

Document: {{.document}}

Please extract the following fields in JSON format:
{{range .keys}}
- {{.}}
{{end}}

Return only valid JSON.`,
			"contact-info": `Extract contact information from the following document:

Document: {{.document}}

Please extract the following fields in JSON format:
{{range .keys}}
- {{.}}
{{end}}

Return only valid JSON.`,
		},
	}
}

func (p *SimplePromptProvider) GetPrompt(tag string, version int) (string, error) {
	if template, exists := p.templates[tag]; exists {
		return template, nil
	}
	return "", fmt.Errorf("template %q not found", tag)
}

func main() {
	fmt.Println("=== Execution Statistics Collection Demo ===")
	fmt.Println("This demo showcases the dry-run capabilities and execution statistics")
	fmt.Println("collection features of the unstruct library.")
	fmt.Println()

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

	// Create Unstructor with logging and custom prompt provider
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	promptProvider := NewSimplePromptProvider()

	unstructor := unstruct.NewWithLogger[TestDocument](client, promptProvider, logger)

	// 1. Test Dry-Run Statistics Collection
	printSectionHeader("1. Testing Dry-Run Statistics Collection")
	assets := []unstruct.Asset{unstruct.NewTextAsset(sampleDoc)}
	stats, err := unstructor.DryRun(context.Background(), assets, unstruct.WithModel("gpt-3.5-turbo"))
	if err != nil {
		fmt.Printf("❌ Dry-run failed: %v\n", err)
		return
	}

	printExecutionStats(stats)

	// 2. Test Enhanced Plan Building with Dry-Run
	printSectionHeader("2. Testing Enhanced Plan Building with Dry-Run")

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
		fmt.Printf("❌ Failed to build plan: %v\n", err)
		return
	}

	// Print the enhanced plan
	fmt.Printf("✅ Plan generated successfully\n")
	fmt.Printf("📋 Expected Models: %v\n", plan.ExpectedModels)
	fmt.Printf("📊 Expected Call Counts: %v\n", plan.ExpectedCallCounts)
	fmt.Printf("💰 Estimated Cost: %.2f abstract units\n", plan.EstCost)

	// 3. Show JSON output
	printSectionHeader("3. Plan JSON Output")
	planJSON, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		fmt.Printf("❌ Error marshaling plan: %v\n", err)
		return
	}
	fmt.Println(string(planJSON))

	// 4. Compare Expected vs Actual (simulation)
	printSectionHeader("4. Expected vs Actual Comparison")
	fmt.Printf("📈 Expected prompt calls: %d\n", len(plan.ExpectedCallCounts))
	fmt.Printf("📊 Actual prompt calls: %d\n", stats.PromptCalls)

	allMatched := true
	for model, expectedCalls := range plan.ExpectedCallCounts {
		actualCalls := stats.ModelCalls[model]
		status := "✅"
		if expectedCalls != actualCalls {
			status = "⚠️ "
			allMatched = false
		}
		fmt.Printf("%s Model %s: expected %d, actual %d\n", status, model, expectedCalls, actualCalls)
	}

	if allMatched {
		fmt.Printf("\n🎉 All model call counts match expectations!\n")
	} else {
		fmt.Printf("\n⚠️  Some model call counts don't match expectations\n")
	}

	// 5. Token Analysis
	printSectionHeader("5. Token Usage Analysis")
	printTokenAnalysis(stats)
}

func printSectionHeader(title string) {
	fmt.Printf("\n%s\n", title)
	fmt.Println(strings.Repeat("-", len(title)))
}

func printExecutionStats(stats *unstruct.ExecutionStats) {
	fmt.Printf("📋 Execution Statistics:\n")
	fmt.Printf("  • Prompt Calls: %d\n", stats.PromptCalls)
	fmt.Printf("  • Prompt Groups: %d\n", stats.PromptGroups)
	fmt.Printf("  • Fields Extracted: %d\n", stats.FieldsExtracted)
	fmt.Printf("  • Total Input Tokens: %d\n", stats.TotalInputTokens)
	fmt.Printf("  • Total Output Tokens: %d\n", stats.TotalOutputTokens)
	fmt.Printf("  • Model Calls: %v\n", stats.ModelCalls)

	fmt.Printf("\n📊 Group Details:\n")
	for i, group := range stats.GroupDetails {
		fmt.Printf("  Group %d:\n", i+1)
		fmt.Printf("    🏷️  Prompt: %s\n", group.PromptName)
		fmt.Printf("    🤖 Model: %s\n", group.Model)
		fmt.Printf("    📝 Fields: %v\n", group.Fields)
		fmt.Printf("    ⬇️  Input Tokens: %d\n", group.InputTokens)
		fmt.Printf("    ⬆️  Output Tokens: %d\n", group.OutputTokens)
		if group.ParentPath != "" {
			fmt.Printf("    📂 Parent Path: %s\n", group.ParentPath)
		}
		fmt.Println()
	}
}

func printTokenAnalysis(stats *unstruct.ExecutionStats) {
	fmt.Printf("🔢 Token Breakdown:\n")
	fmt.Printf("  • Total Input:  %d tokens\n", stats.TotalInputTokens)
	fmt.Printf("  • Total Output: %d tokens\n", stats.TotalOutputTokens)
	fmt.Printf("  • Total Usage:  %d tokens\n", stats.TotalInputTokens+stats.TotalOutputTokens)

	if len(stats.GroupDetails) > 0 {
		fmt.Printf("\n📈 Per-Group Analysis:\n")
		for i, group := range stats.GroupDetails {
			ratio := float64(group.OutputTokens) / float64(group.InputTokens)
			fmt.Printf("  Group %d (%s):\n", i+1, group.PromptName)
			fmt.Printf("    Input:Output ratio: 1:%.2f\n", ratio)
			fmt.Printf("    Efficiency: %.1f%% output\n", (float64(group.OutputTokens)/float64(group.InputTokens+group.OutputTokens))*100)
		}
	}
}

func demoWithoutAPI(sampleDoc string) {
	fmt.Println("🚀 Running in demo mode without API...")
	fmt.Println("This simulates the statistics that would be collected during actual execution.")
	fmt.Println()

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

	printSectionHeader("Demo Statistics")
	printExecutionStats(stats)

	printSectionHeader("Demo Token Analysis")
	printTokenAnalysis(stats)

	fmt.Printf("\n💡 To see real execution statistics, set the GEMINI_API_KEY environment variable.\n")
}
