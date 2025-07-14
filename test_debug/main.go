package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// TestExtract for testing the file processing issue
type TestExtract struct {
	Name        string `json:"name" unstruct:"base"`
	Description string `json:"description" unstruct:"base"`
	Count       int    `json:"count" unstruct:"base"`
}

func main() {
	// Check for API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create genai client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create genai client: %v", err)
	}

	// Simple prompt template
	template := `Extract information from the document:
{{ document }}

Return JSON with fields: name, description, count`

	provider := unstruct.SimplePromptProvider{"base": template}
	extractor := unstruct.New[TestExtract](client, provider)

	// TEST 1: Text Asset (WORKS) ✅
	fmt.Println("=== TEXT ASSET TEST ===")
	testText := "The magic widget is a fantastic device. There are 42 widgets in stock."
	asset := unstruct.NewTextAsset(testText)

	result, err := extractor.Unstruct(ctx, []unstruct.Asset{asset}, unstruct.WithModel("gemini-1.5-pro"))
	if err != nil {
		log.Fatalf("Text asset failed: %v", err)
	}

	fmt.Printf("✅ TEXT ASSET RESULT: %+v\n", *result)

	// TEST 2: File Asset (FAILS) ❌
	fmt.Println("\n=== FILE ASSET TEST ===")
	// Create test file with same content
	testFile := "/tmp/test_issue.txt"
	testContent := "The magic widget is a fantastic device. There are 42 widgets in stock."

	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	fileAsset := unstruct.NewFileAsset(client, testFile)

	result, err = extractor.Unstruct(ctx, []unstruct.Asset{fileAsset}, unstruct.WithModel("gemini-1.5-pro"))
	if err != nil {
		log.Fatalf("File asset failed: %v", err)
	}

	fmt.Printf("❌ FILE ASSET RESULT: %+v\n", *result)

	// This should extract the same data but will likely fail
	if result.Name == "Gemini Files API" {
		fmt.Println("🐛 BUG CONFIRMED: File asset returns upload metadata instead of content")
		fmt.Printf("Expected to extract 'magic widget' but got '%s'\n", result.Name)
		fmt.Println("This proves the AI is seeing upload info, not file content")
	} else {
		fmt.Println("✅ File asset working correctly!")
	}

	// TEST 3: Manual file upload to understand the correct API usage
	fmt.Println("\n=== MANUAL FILE UPLOAD TEST ===")
	fmt.Println("Uploading file manually to understand correct API usage...")

	// Upload file manually to see the correct approach
	file, err := client.Files.UploadFromPath(ctx, testFile, &genai.UploadFileConfig{
		MIMEType:    "text/plain",
		DisplayName: "Test Document",
	})
	if err != nil {
		log.Fatalf("Manual upload failed: %v", err)
	}

	fmt.Printf("Manual upload successful! File URI: %s\n", file.URI)
	fmt.Printf("File Name: %s\n", file.Name)

	// Now let's try to understand how to properly reference this file in content
	// The key question: How do we create genai.Content that references an uploaded file?

	fmt.Println("\n=== TESTING CONTENT CREATION WITH FILE ===")

	// Let's try different approaches to see what works
	// Approach 1: Text that mentions the file URI (current broken approach)
	prompt1 := fmt.Sprintf("Analyze the uploaded file: %s\n\nExtract: name, description, count", file.URI)
	content1 := genai.NewContentFromText(prompt1, genai.RoleUser)

	fmt.Printf("Approach 1 - Text mentioning URI: %s\n", prompt1[:50]+"...")

	// Test this approach
	resp1, err := client.Models.GenerateContent(ctx, "gemini-1.5-pro", []*genai.Content{content1}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		log.Printf("Approach 1 failed: %v", err)
	} else if len(resp1.Candidates) > 0 && resp1.Candidates[0].Content != nil {
		fmt.Printf("Approach 1 response: %s\n", resp1.Candidates[0].Content.Parts[0].Text[:100]+"...")
	}

	// Now we need to find the correct way to create content with file data
	// This is where the fix needs to be implemented
	fmt.Println("\n🔍 ANALYSIS:")
	fmt.Println("- File upload works correctly")
	fmt.Println("- Text-only prompts mentioning file URI don't work")
	fmt.Println("- Need to find genai.NewContentFromFileData() or similar")
	fmt.Println("- The unstruct library needs to create proper file data parts")
}
