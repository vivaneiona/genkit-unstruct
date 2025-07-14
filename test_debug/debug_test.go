package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// Simple test struct
type TestData struct {
	Content     string `json:"content" unstruct:"extract"`
	Description string `json:"description" unstruct:"extract"`
}

func main() {
	// Check for API key
	if os.Getenv("GEMINI_API_KEY") == "" {
		fmt.Println("⚠️  GEMINI_API_KEY not set - please set it to test the file processing fix")
		return
	}

	fmt.Println("🔧 Testing file processing fix with debug info...")
	testFileProcessingWithDebug()
}

func testFileProcessingWithDebug() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create genai client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create genai client: %v", err)
	}

	// Create more specific prompt for debugging
	provider := unstruct.SimplePromptProvider{
		"extract": `Analyze the provided document and extract:
{{ document }}

Extract the following information as JSON:
- content: The main textual content found in the document
- description: A brief description of what this document contains

Return only valid JSON with the fields: content, description`,
	}

	extractor := unstruct.New[TestData](client, provider)

	// Test file asset
	fmt.Println("=== Testing FileAsset with Debug ===")

	// Create a test file
	testFile := "/tmp/unstruct_debug_test.txt"
	testContent := "IMPORTANT CONTENT: This document contains critical information about Project Phoenix. The project involves developing advanced AI systems for autonomous vehicles."
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	fmt.Printf("📄 Test file content: %s\n", testContent)

	// Test with file asset
	fileAsset := unstruct.NewFileAsset(client, testFile)

	// First, let's test what messages are created
	fmt.Println("\n🔍 Debugging FileAsset.CreateMessages()...")

	// Create a simple logger for testing
	log := slog.Default()
	messages, err := fileAsset.CreateMessages(ctx, log)
	if err != nil {
		log.Fatalf("Failed to create messages: %v", err)
	}

	fmt.Printf("📨 Messages created: %d\n", len(messages))
	for i, msg := range messages {
		fmt.Printf("  Message %d: Role=%s, Parts=%d\n", i, msg.Role, len(msg.Parts))
		for j, part := range msg.Parts {
			fmt.Printf("    Part %d: Type=%s", j, part.Type)
			switch part.Type {
			case "text":
				fmt.Printf(", Text=%s", part.Text[:min(50, len(part.Text))]+"...")
			case "file":
				fmt.Printf(", FileURI=%s, MimeType=%s", part.FileURI, part.MimeType)
			}
			fmt.Println()
		}
	}

	// Now test the full extraction
	fmt.Println("\n🤖 Running full extraction...")
	result, err := extractor.Unstruct(ctx, []unstruct.Asset{fileAsset}, unstruct.WithModel("gemini-1.5-pro"))
	if err != nil {
		log.Printf("FileAsset extraction failed: %v", err)
		return
	}

	fmt.Printf("📊 Extraction result: %+v\n", *result)

	// Analyze the results
	if result.Content != "" && result.Description != "" {
		fmt.Println("🎉 SUCCESS: Both content and description extracted correctly!")
		fmt.Println("✅ The file processing fix is working perfectly!")
	} else if result.Content == "" && result.Description != "" {
		fmt.Println("⚠️  PARTIAL SUCCESS: Description extracted but content is empty")
		fmt.Println("💡 The file is being processed (not getting upload metadata) but content field mapping might need adjustment")
	} else if result.Content != "" && result.Description == "" {
		fmt.Println("⚠️  PARTIAL SUCCESS: Content extracted but description is empty")
	} else {
		fmt.Println("❌ ISSUE: Neither content nor description extracted properly")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
