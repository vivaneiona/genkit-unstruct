package main

import (
	"context"
	"fmt"
	"log"
	"os"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// Example showing group-based extraction
type Person struct {
	Name string `json:"name" unstruct:"group/basic-info"`
	Age  int    `json:"age"  unstruct:"group/basic-info"`
	City string `json:"city" unstruct:"group/basic-info"`
}

// Example with mixed group and direct tags
type DetailedPerson struct {
	Name    string `json:"name" unstruct:"group/basic-info"`
	Age     int    `json:"age"  unstruct:"group/basic-info"`
	Address string `json:"address" unstruct:"address,gemini-1.5-pro"`
	Email   string `json:"email" unstruct:"group/contact-info"`
	Phone   string `json:"phone" unstruct:"group/contact-info"`
}

func main() {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Create a simple prompt provider for demonstration
	prompts, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
	)
	if err != nil {
		log.Fatal("Failed to create Stick prompt provider:", err)
	}

	// Example 1: Simple group usage
	fmt.Println("=== Example 1: Simple Group Usage ===")
	u1 := unstruct.New[Person](client, prompts)
	
	// Dry run to show the plan without making API calls
	stats1, err := u1.DryRunFromText(ctx, "John Doe, 25 years old, lives in New York",
		unstruct.WithModel("gemini-1.5-flash"), // fallback model
		unstruct.WithGroup("basic-info", "basic", "gemini-2.0-flash"),
	)
	if err != nil {
		log.Printf("Dry run failed: %v", err)
	} else {
		fmt.Printf("Groups created: %d\n", stats1.PromptCalls)
		fmt.Printf("Expected input tokens: %d\n", stats1.TotalInputTokens)
		fmt.Printf("Expected output tokens: %d\n", stats1.TotalOutputTokens)
	}

	// Example 2: Mixed groups and direct tags
	fmt.Println("\n=== Example 2: Mixed Groups and Direct Tags ===")
	u2 := unstruct.New[DetailedPerson](client, prompts)
	
	// Dry run with multiple groups
	stats2, err := u2.DryRunFromText(ctx, "John Doe, 25, 123 Main St, john@example.com, 555-1234",
		unstruct.WithModel("gemini-1.5-flash"), // fallback model
		unstruct.WithGroup("basic-info", "basic", "gemini-2.0-flash"),
		unstruct.WithGroup("contact-info", "contact", "gemini-1.5-pro"),
	)
	if err != nil {
		log.Printf("Dry run failed: %v", err)
	} else {
		fmt.Printf("Groups created: %d\n", stats2.PromptCalls)
		fmt.Printf("Expected input tokens: %d\n", stats2.TotalInputTokens)
		fmt.Printf("Expected output tokens: %d\n", stats2.TotalOutputTokens)
	}

	// Example 3: Show explanation
	fmt.Println("\n=== Example 3: Detailed Explanation ===")
	explanation, err := u2.ExplainFromText(ctx, "John Doe, 25, 123 Main St, john@example.com, 555-1234",
		unstruct.WithModel("gemini-1.5-flash"), // fallback model
		unstruct.WithGroup("basic-info", "basic", "gemini-2.0-flash"),
		unstruct.WithGroup("contact-info", "contact", "gemini-1.5-pro"),
	)
	if err != nil {
		log.Printf("Explanation failed: %v", err)
	} else {
		fmt.Println(explanation)
	}
}
