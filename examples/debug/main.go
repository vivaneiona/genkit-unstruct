package main

import (
	"fmt"

	unstruct "github.com/vivaneiona/genkit-unstruct"
)

func main() {
	// Create a plan builder
	builder := unstruct.NewPlanBuilder()

	// Define a simple schema for debugging
	schema := map[string]interface{}{
		"fields": []string{"Name", "Age"},
	}

	// Configure models for different fields
	modelConfig := map[string]string{
		"Name": "gpt-3.5-turbo",
		"Age":  "gpt-3.5-turbo",
	}

	// Build the plan
	builder.WithSchema(schema).WithModelConfig(modelConfig)

	// Debug: Get the plan and examine its structure
	plan, err := builder.Explain()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print plan structure manually
	fmt.Println("=== Plan Structure Debug ===")
	printNode(plan, "", true)

	fmt.Println("\n=== Official Text Format ===")
	textPlan, err := builder.ExplainPretty(unstruct.FormatText)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(textPlan)
}

func printNode(node *unstruct.PlanNode, prefix string, isLast bool) {
	// Choose the appropriate tree connector
	connector := "├─ "
	if isLast {
		connector = "└─ "
	}
	if prefix == "" {
		connector = ""
	}

	fmt.Printf("%s%s%s (children: %d)\n", prefix, connector, node.Type, len(node.Children))

	// Format children
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "   "
		} else {
			childPrefix += "│  "
		}
	}

	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		printNode(child, childPrefix, isLastChild)
	}
}
