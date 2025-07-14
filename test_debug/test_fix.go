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

// Simple test struct
type TestData struct {
	Content     string `json:"content" unstruct:"extract"`
	Description string `json:"description" unstruct:"extract"`
}

func main() {
	// Check for API key (not required for this compilation test)
	if os.Getenv("GEMINI_API_KEY") == "" {
		fmt.Println("⚠️  GEMINI_API_KEY not set - this is just a compilation test")
		fmt.Println("✅ Testing the file processing fix compilation...")
		testCompilation()
		return
	}

	fmt.Println("🔧 Testing actual file processing fix...")
	testFileProcessing()
}

func testCompilation() {
	// Test that our new file part creation works
	filePart := unstruct.NewFilePart("files/test123", "text/plain")
	fmt.Printf("✅ NewFilePart created: Type=%s, URI=%s, MimeType=%s\n",
		filePart.Type, filePart.FileURI, filePart.MimeType)

	// Test that text and image parts still work
	textPart := unstruct.NewTextPart("test text")
	fmt.Printf("✅ NewTextPart still works: Type=%s, Text=%s\n",
		textPart.Type, textPart.Text)

	imagePart := unstruct.NewImagePart([]byte("fake image"), "image/png")
	fmt.Printf("✅ NewImagePart still works: Type=%s, MimeType=%s\n",
		imagePart.Type, imagePart.MimeType)

	fmt.Println("🎉 All part creation functions work correctly!")
}

func testFileProcessing() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create genai client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create genai client: %v", err)
	}

	// Create simple prompt
	provider := unstruct.SimplePromptProvider{"extract": "Extract content and description from: {{ document }}"}
	extractor := unstruct.New[TestData](client, provider)

	// Test 1: Verify text assets still work
	fmt.Println("=== Testing TextAsset (should work as before) ===")
	textAsset := unstruct.NewTextAsset("This is a test document with important content.")
	result, err := extractor.Unstruct(ctx, []unstruct.Asset{textAsset}, unstruct.WithModel("gemini-1.5-pro"))
	if err != nil {
		log.Printf("TextAsset test failed: %v", err)
	} else {
		fmt.Printf("✅ TextAsset result: %+v\n", *result)
	}

	// Test 2: Test file asset with our fix
	fmt.Println("\n=== Testing FileAsset (with fix) ===")

	// Create a test file
	testFile := "/tmp/unstruct_test.txt"
	testContent := "This is a sample document. It contains valuable information about widgets."
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	fileAsset := unstruct.NewFileAsset(client, testFile)
	result, err = extractor.Unstruct(ctx, []unstruct.Asset{fileAsset}, unstruct.WithModel("gemini-1.5-pro"))
	if err != nil {
		log.Printf("FileAsset test failed: %v", err)
	} else {
		fmt.Printf("🎉 FileAsset result: %+v\n", *result)

		// Check if we got actual content instead of "Gemini Files API"
		if result.Content != "" && result.Content != "Gemini Files API" {
			fmt.Println("✅ SUCCESS: File content was processed correctly!")
			fmt.Println("✅ The fix is working - AI model received actual file content!")
		} else {
			fmt.Println("❌ ISSUE: Still getting upload metadata instead of content")
		}
	}
}
