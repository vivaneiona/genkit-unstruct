package main

import (
	"encoding/json"
	"fmt"

	unstruct "github.com/vivaneiona/genkit-unstruct"
)

func main() {
	// Create a schema with different models for different fields
	schema := map[string]interface{}{
		"fields": []string{"name", "age", "address", "email"},
	}

	// Configure models for different fields
	modelConfig := map[string]string{
		"name":    "gpt-4o",
		"age":     "gpt-3.5-turbo",
		"address": "claude-3-sonnet",
		"email":   "gpt-4o", // Same as name, should not duplicate in expected models
	}

	// Build the plan
	builder := unstruct.NewPlanBuilder().
		WithSchema(schema).
		WithModelConfig(modelConfig)

	plan, err := builder.Explain()
	if err != nil {
		fmt.Printf("Error building plan: %v\n", err)
		return
	}

	// Print summary information
	fmt.Println("=== Plan Summary ===")
	fmt.Printf("Expected Models: %v\n", plan.ExpectedModels)
	fmt.Printf("Expected Call Counts: %v\n", plan.ExpectedCallCounts)

	// Print the plan structure in JSON for verification
	fmt.Println("\n=== Full Plan (JSON) ===")
	planJSON, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling plan: %v\n", err)
		return
	}
	fmt.Println(string(planJSON))
}
