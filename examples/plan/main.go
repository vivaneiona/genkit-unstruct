package main

import (
	"fmt"
	"log"

	unstruct "github.com/vivaneiona/genkit-unstruct"
)

func main() {
	fmt.Println("=== Unstructor Execution Plan Demo ===")

	// Create a plan builder
	builder := unstruct.NewPlanBuilder()

	// Define a sample schema for extracting information from a resume
	schema := map[string]interface{}{
		"fields":      []string{"Name", "Email", "Phone", "Experience", "Education", "Skills"},
		"description": "Extract structured information from a resume",
	}

	// Configure models for different fields
	modelConfig := map[string]string{
		"Name":       "gpt-3.5-turbo",
		"Email":      "gpt-3.5-turbo",
		"Phone":      "gpt-3.5-turbo",
		"Experience": "gpt-4", // Use more powerful model for complex extraction
		"Education":  "gpt-4",
		"Skills":     "gpt-3.5-turbo",
	}

	// Build the plan
	builder.WithSchema(schema).WithModelConfig(modelConfig)

	// Example 1: Basic explanation with abstract costs
	fmt.Println("1. Basic Execution Plan (Abstract Costs)")

	textPlan, err := builder.ExplainPretty(unstruct.FormatText)
	if err != nil {
		log.Fatalf("Error generating text plan: %v", err)
	}
	fmt.Println(textPlan)

	// Example 2: Explanation with real cost estimates
	fmt.Println("\n2. Execution Plan with Real Cost Estimates")

	// Use default pricing but can be customized
	pricing := unstruct.DefaultModelPricing()

	textPlanWithCosts, err := builder.ExplainPrettyWithCosts(unstruct.FormatText, pricing)
	if err != nil {
		log.Fatalf("Error generating text plan with costs: %v", err)
	}
	fmt.Println(textPlanWithCosts)

	// Example 3: JSON format for programmatic use
	fmt.Println("\n3. JSON Format (for programmatic consumption)")

	jsonPlan, err := builder.ExplainPrettyWithCosts(unstruct.FormatJSON, pricing)
	if err != nil {
		log.Fatalf("Error generating JSON plan: %v", err)
	}
	fmt.Println(jsonPlan)

	// Example 4: Graphviz DOT format for visualization
	fmt.Println("\n4. Graphviz DOT Format (for visualization)")

	dotPlan, err := builder.ExplainPretty(unstruct.FormatGraphviz)
	if err != nil {
		log.Fatalf("Error generating DOT plan: %v", err)
	}
	fmt.Println(dotPlan)
	fmt.Println("\n# Copy the above DOT code to https://dreampuf.github.io/GraphvizOnline/ for visualization")

	// Example 5: Demonstrate cost analysis
	fmt.Println("\n5. Cost Analysis Summary")

	plan, err := builder.ExplainWithCosts(pricing)
	if err != nil {
		log.Fatalf("Error generating plan: %v", err)
	}

	totalEstimatedCost := plan.EstCost
	totalActualCost := 0.0
	totalTokens := 0
	promptCallCount := 0

	// Traverse the plan to collect statistics
	var collectStats func(*unstruct.PlanNode)
	collectStats = func(node *unstruct.PlanNode) {
		if node.Type == unstruct.PromptCallType {
			promptCallCount++
			totalTokens += node.InputTokens + node.OutputTokens
			if node.ActCost != nil {
				totalActualCost += *node.ActCost
			}
		}
		for _, child := range node.Children {
			collectStats(child)
		}
	}

	collectStats(plan)

	fmt.Printf("Total fields to extract: %d\n", len(plan.Fields))
	fmt.Printf("Total prompt calls: %d\n", promptCallCount)
	fmt.Printf("Total estimated tokens: %d\n", totalTokens)
	fmt.Printf("Total abstract cost units: %.2f\n", totalEstimatedCost)
	fmt.Printf("Total estimated USD cost: $%.6f\n", totalActualCost)

	// Cost breakdown by model
	fmt.Println("\nCost breakdown by model:")
	modelCosts := make(map[string]float64)
	modelTokens := make(map[string]int)

	var collectModelStats func(*unstruct.PlanNode)
	collectModelStats = func(node *unstruct.PlanNode) {
		if node.Type == unstruct.PromptCallType && node.Model != "" {
			if node.ActCost != nil {
				modelCosts[node.Model] += *node.ActCost
			}
			modelTokens[node.Model] += node.InputTokens + node.OutputTokens
		}
		for _, child := range node.Children {
			collectModelStats(child)
		}
	}

	collectModelStats(plan)

	for model, cost := range modelCosts {
		tokens := modelTokens[model]
		fmt.Printf("  %s: $%.6f (%d tokens)\n", model, cost, tokens)
	}

	// Example 6: HTML format
	fmt.Println("\n6. Generating HTML Report")

	htmlPlan, err := builder.ExplainPretty(unstruct.FormatHTML)
	if err != nil {
		log.Fatalf("Error generating HTML plan: %v", err)
	}

	// Write to file (in a real application)
	fmt.Println("HTML report generated (would write to file in real application)")
	fmt.Printf("HTML length: %d characters\n", len(htmlPlan))

	// Example 7: Token estimation utilities
	fmt.Println("\n7. Token Estimation Utilities")

	sampleText := "John Doe is a software engineer with 5 years of experience in Go, Python, and JavaScript. He graduated from MIT with a degree in Computer Science."

	tokensFromText := unstruct.EstimateTokensFromText(sampleText)
	tokensFromWords := unstruct.EstimateTokensFromWords(24) // approximate word count

	fmt.Printf("Sample text: %s\n", sampleText)
	fmt.Printf("Estimated tokens from text length: %d\n", tokensFromText)
	fmt.Printf("Estimated tokens from word count: %d\n", tokensFromWords)

	fmt.Println("\n=== Demo Complete ===")
}

// Example 8: Demonstrate Dry-Run Execution Statistics (if we had an actual Unstructor)
func demonstrateDryRunStats() {
	fmt.Println("\n8. Dry-Run Execution Statistics Demo")
	fmt.Println("Note: This would work with an actual Unstructor instance:")

	// Example of what dry-run statistics would look like
	fmt.Println(`
	Sample Dry-Run Output:
	----------------------
	Prompt Calls: 6
	Prompt Groups: 4 
	Fields Extracted: 6
	Total Input Tokens: 2200
	Total Output Tokens: 180
	Model Calls: {gpt-3.5-turbo: 4, gpt-4: 2}
	
	Group Details:
	  1. prompt-group-1 (gpt-3.5-turbo): [Name, Email, Phone] -> 1200 tokens in, 60 tokens out
	  2. prompt-group-2 (gpt-4): [Experience] -> 500 tokens in, 60 tokens out  
	  3. prompt-group-3 (gpt-4): [Education] -> 450 tokens in, 50 tokens out
	  4. prompt-group-4 (gpt-3.5-turbo): [Skills] -> 50 tokens in, 10 tokens out
	
	Expected vs Actual Comparison:
	✓ All prompt calls matched expected count
	✓ Model distribution as planned
	✓ Token estimates within acceptable range
	`)

	fmt.Println("\n=== Demo Complete ===")
}

// Utility function to repeat a string (since Go doesn't have built-in string repetition)
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
