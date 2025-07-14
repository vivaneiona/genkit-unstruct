package main

import (
	"fmt"
	"log"

	unstruct "github.com/vivaneiona/genkit-unstruct"
)

func main() {
	// Example from the header documentation
	schema := map[string]interface{}{
		"fields": []string{"name", "email", "company"},
	}

	// Basic usage example
	fmt.Println("=== Basic Usage Example ===")
	_, err := unstruct.NewPlanBuilder().
		WithSchema(schema).
		Explain()
	if err != nil {
		log.Fatal(err)
	}

	// Format as text
	textPlan, err := unstruct.NewPlanBuilder().
		WithSchema(schema).
		ExplainPretty(unstruct.FormatText)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(textPlan)

	// Cost estimation example
	fmt.Println("\n=== Cost Estimation Example ===")
	pricing := unstruct.DefaultModelPricing()

	costPlan, err := unstruct.NewPlanBuilder().
		WithSchema(schema).
		ExplainPrettyWithCosts(unstruct.FormatText, pricing)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(costPlan)

	// Advanced configuration example
	fmt.Println("\n=== Advanced Configuration Example ===")
	modelConfig := map[string]string{
		"email": "gpt-4o-mini",
		"name":  "gemini-1.5-flash",
	}

	promptConfig := map[string]interface{}{
		"email": "EmailExtractionPrompt",
		"name":  "NameExtractionPrompt",
	}

	advancedPlan, err := unstruct.NewPlanBuilder().
		WithSchema(schema).
		WithModelConfig(modelConfig).
		WithPromptConfig(promptConfig).
		ExplainWithCosts(pricing)
	if err != nil {
		log.Fatal(err)
	}

	// Format as JSON
	fmt.Println("\n=== JSON Format Example ===")
	jsonOutput, err := unstruct.NewPlanBuilder().
		WithSchema(schema).
		ExplainPretty(unstruct.FormatJSON)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(jsonOutput)

	fmt.Printf("\nAdvanced plan estimated cost: %.2f\n", advancedPlan.EstCost)
	fmt.Printf("Number of expected models: %d\n", len(advancedPlan.ExpectedModels))
	fmt.Printf("Expected models: %v\n", advancedPlan.ExpectedModels)
}
