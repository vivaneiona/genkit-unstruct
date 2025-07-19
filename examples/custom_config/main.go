package main

import (
	"fmt"
	"log"

	unstruct "github.com/vivaneiona/genkit-unstruct"
)

func main() {
	fmt.Println("=== Custom Token and Cost Configuration Demo ===")

	// Define a sample schema
	schema := map[string]interface{}{
		"fields": []string{"name", "email", "company", "description"},
	}

	// Create a plan builder with default settings
	defaultBuilder := unstruct.NewPlanBuilder().WithSchema(schema)

	// Generate plan with default settings
	defaultPlan, err := defaultBuilder.Explain()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Default Plan Cost: %.2f\n", defaultPlan.EstCost)

	// Create custom token estimation configuration
	customTokenConfig := unstruct.TokenEstimationConfig{
		CharsPerToken:      3,   // More aggressive token estimation
		TokensPerWordRatio: 1.5, // Higher tokens per word
		BasePromptTokens:   100, // Larger base prompt
		DocumentTokens:     500, // More document context
		SchemaBaseTokens:   50,  // Higher schema overhead
		TokensPerField:     10,  // More tokens per field
	}

	// Create custom cost calculation configuration
	customCostConfig := unstruct.CostCalculationConfig{
		SchemaAnalysisBaseCost: 2.0,  // Higher schema analysis cost
		SchemaAnalysisPerField: 1.0,  // Higher per-field cost
		PromptCallBaseCost:     5.0,  // Higher prompt call cost
		PromptCallTokenFactor:  0.02, // Higher token factor
		MergeFragmentsBaseCost: 1.0,  // Higher merge cost
		MergeFragmentsPerField: 0.2,  // Higher merge per-field cost
		TransformCost:          3.0,  // Higher transform cost
		DefaultNodeCost:        2.0,  // Higher default cost
	}

	// Create a plan builder with custom settings
	customBuilder := unstruct.NewPlanBuilder().
		WithSchema(schema).
		WithTokenConfig(customTokenConfig).
		WithCostConfig(customCostConfig)

	// Generate plan with custom settings
	customPlan, err := customBuilder.Explain()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Custom Plan Cost: %.2f\n", customPlan.EstCost)
	fmt.Printf("Cost Difference: %.2f (%.1f%% increase)\n",
		customPlan.EstCost-defaultPlan.EstCost,
		((customPlan.EstCost-defaultPlan.EstCost)/defaultPlan.EstCost)*100)

	// Demonstrate custom token estimation
	sampleText := "This is a sample document with multiple words for token estimation testing."

	defaultTokens := unstruct.EstimateTokensFromText(sampleText)
	customTokens := customBuilder.EstimateTokensFromTextCustom(sampleText)

	fmt.Printf("\nToken Estimation for: %q\n", sampleText)
	fmt.Printf("Default estimation: %d tokens\n", defaultTokens)
	fmt.Printf("Custom estimation: %d tokens\n", customTokens)

	// Show configuration examples
	fmt.Println("\n=== Configuration Examples ===")

	// Example 1: Conservative estimation (for budget planning)
	conservativeConfig := unstruct.TokenEstimationConfig{
		CharsPerToken:      3,    // More tokens per character
		TokensPerWordRatio: 1.8,  // More tokens per word
		BasePromptTokens:   200,  // Larger safety margin
		DocumentTokens:     1000, // More document overhead
		SchemaBaseTokens:   100,  // Higher schema costs
		TokensPerField:     20,   // More tokens per field
	}

	// Example 2: Optimistic estimation (for quick estimates)
	optimisticConfig := unstruct.TokenEstimationConfig{
		CharsPerToken:      5,   // Fewer tokens per character
		TokensPerWordRatio: 1.0, // Exact word-to-token ratio
		BasePromptTokens:   25,  // Minimal prompt overhead
		DocumentTokens:     100, // Minimal document overhead
		SchemaBaseTokens:   10,  // Lower schema costs
		TokensPerField:     3,   // Fewer tokens per field
	}

	conservativePlan, _ := unstruct.NewPlanBuilder().
		WithSchema(schema).
		WithTokenConfig(conservativeConfig).
		Explain()

	optimisticPlan, _ := unstruct.NewPlanBuilder().
		WithSchema(schema).
		WithTokenConfig(optimisticConfig).
		Explain()

	fmt.Printf("Conservative estimate: %.2f cost units\n", conservativePlan.EstCost)
	fmt.Printf("Default estimate: %.2f cost units\n", defaultPlan.EstCost)
	fmt.Printf("Optimistic estimate: %.2f cost units\n", optimisticPlan.EstCost)
}
