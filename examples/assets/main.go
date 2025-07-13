package main

import (
"context"
"fmt"
"log"

unstruct "github.com/vivaneiona/genkit-unstruct"
)

// Example struct to extract
type Document struct {
	Title       string `json:"title" prompt:"basic"`
	Description string `json:"description" prompt:"basic"`
	Category    string `json:"category" prompt:"basic"`
}

// Simple prompt provider for the example
type simplePrompts struct{}

func (s simplePrompts) GetPrompt(tag string, version int) (string, error) {
	prompts := map[string]string{
		"basic": "Extract the title, description, and category from this document: {{.Document}}",
	}
	if prompt, ok := prompts[tag]; ok {
		return prompt, nil
	}
	return "", fmt.Errorf("prompt not found: %s", tag)
}

func main() {
	// Create unstructor
	u := unstruct.NewForTesting[Document](simplePrompts{})
	
	// Example 1: Text-only document
	fmt.Println("=== Text Document Example ===")
	textDoc := "Technical Report: Advanced AI Systems. This document describes the implementation of machine learning algorithms for natural language processing. Category: Technology"
	
	textAsset := unstruct.NewTextAsset(textDoc)
	assets := []unstruct.Asset{textAsset}
	
	result, err := u.Unstruct(
context.Background(),
		assets,
		unstruct.WithModel("gemini-1.5-flash"),
	)
	if err != nil {
		log.Printf("Error extracting from text: %v", err)
	} else {
		fmt.Printf("Title: %s\n", result.Title)
		fmt.Printf("Description: %s\n", result.Description)
		fmt.Printf("Category: %s\n", result.Category)
	}
	
	fmt.Println("\n=== Dry Run Example ===")
	// Example 2: Dry run for cost estimation
	stats, err := u.DryRun(
context.Background(),
		assets,
		unstruct.WithModel("gemini-1.5-flash"),
	)
	if err != nil {
		log.Printf("Error in dry run: %v", err)
	} else {
		fmt.Printf("Estimated prompt calls: %d\n", stats.PromptCalls)
		fmt.Printf("Estimated input tokens: %d\n", stats.TotalInputTokens)
		fmt.Printf("Estimated output tokens: %d\n", stats.TotalOutputTokens)
	}
}
