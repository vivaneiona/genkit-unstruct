package main

import (
	"context"
	"fmt"
	"log"
	"os"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// Simple structure for real extraction test
type PersonInfo struct {
	Name    string `json:"name" unstruct:"group/basic"`
	Age     int    `json:"age" unstruct:"group/basic"`
	City    string `json:"city" unstruct:"group/basic"`
	Job     string `json:"job" unstruct:"group/professional"`
	Company string `json:"company" unstruct:"group/professional"`
}

func main() {
	fmt.Println("Real Groups Extraction Test")
	fmt.Println("==========================")

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	prompts, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
	)
	if err != nil {
		log.Fatal("Failed to create prompt provider:", err)
	}

	u := unstruct.New[PersonInfo](client, prompts)

	// Test data
	testData := "Sarah Thompson is 28 years old and lives in Seattle. She works as a Data Scientist at Microsoft Corporation."

	fmt.Printf("Input: %s\n\n", testData)

	// First show the plan
	fmt.Println("=== Execution Plan ===")
	explanation, err := u.ExplainFromText(ctx, testData,
		unstruct.WithModel("gemini-1.5-flash"),
		unstruct.WithGroup("basic", "basic", "gemini-2.0-flash"),
		unstruct.WithGroup("professional", "work-info", "gemini-1.5-pro"),
	)
	if err != nil {
		log.Printf("Explanation failed: %v", err)
	} else {
		fmt.Println(explanation)
	}

	// Now perform the actual extraction
	fmt.Println("\n=== Actual Extraction ===")
	result, err := u.UnstructFromText(ctx, testData,
		unstruct.WithModel("gemini-1.5-flash"),
		unstruct.WithGroup("basic", "basic", "gemini-2.0-flash"),
		unstruct.WithGroup("professional", "work-info", "gemini-1.5-pro"),
	)
	if err != nil {
		log.Printf("Extraction failed: %v", err)
		return
	}

	fmt.Printf("Extracted data:\n")
	fmt.Printf("  Name: %s\n", result.Name)
	fmt.Printf("  Age: %d\n", result.Age)
	fmt.Printf("  City: %s\n", result.City)
	fmt.Printf("  Job: %s\n", result.Job)
	fmt.Printf("  Company: %s\n", result.Company)

	fmt.Println("\nGroup-based extraction completed successfully!")
	fmt.Println("Notice how:")
	fmt.Println("- Basic info (name, age, city) was extracted with one API call using 'basic' template")
	fmt.Println("- Professional info (job, company) was extracted with another API call using 'work-info' template")
	fmt.Println("- Different models were used for different types of information")
}
