package main

import (
	"context"
	"fmt"
	"log"

	unstruct "github.com/vivaneiona/genkit-unstruct"
)

// TestStruct for demonstration
type TestStruct struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
}

func main() {
	// Create a simple prompt provider for testing
	prompts := unstruct.SimplePromptProvider{
		"default": "Extract the following fields: {{.Keys}}",
	}

	// Note: In a real scenario, you'd pass a proper client
	// For this test, we just want to verify the Explain method works
	unstructor := unstruct.NewWithLogger[TestStruct](nil, prompts, nil)

	// Create test assets
	assets := []unstruct.Asset{
		unstruct.NewTextAsset("John Doe works at Acme Corp. His email is john@acme.com"),
	}

	// Test the new Explain method
	fmt.Println("=== Testing Unstructor.Explain() Method ===")

	explanation, err := unstructor.Explain(
		context.Background(),
		assets,
		unstruct.WithModel("gpt-3.5-turbo"),
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Execution Plan:")
	fmt.Println(explanation)

	// Test the convenience method ExplainFromText
	fmt.Println("\n=== Testing Unstructor.ExplainFromText() Convenience Method ===")

	textExplanation, err := unstructor.ExplainFromText(
		context.Background(),
		"John Doe works at Acme Corp. His email is john@acme.com",
		unstruct.WithModel("gpt-3.5-turbo"),
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Println("Text-based Execution Plan:")
	fmt.Println(textExplanation)
}
