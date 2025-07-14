package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// DocumentMetadata represents basic document information
type DocumentMetadata struct {
	Title       string `json:"title" unstruct:"basic"`
	Description string `json:"description" unstruct:"basic"`
	Category    string `json:"category" unstruct:"basic"`
	Author      string `json:"author" unstruct:"person"`
	Date        string `json:"date" unstruct:"basic"`
	Version     string `json:"version" unstruct:"basic"`
}

// ProjectInfo represents project-specific information extracted from documents
type ProjectInfo struct {
	ProjectCode string  `json:"projectCode" unstruct:"project"`
	ProjectName string  `json:"projectName" unstruct:"project"`
	Budget      float64 `json:"budget" unstruct:"project"`
	Currency    string  `json:"currency" unstruct:"project"`
	StartDate   string  `json:"startDate" unstruct:"project"`
	EndDate     string  `json:"endDate" unstruct:"project"`
	Status      string  `json:"status" unstruct:"project"`
	Priority    string  `json:"priority" unstruct:"project"`
	ProjectLead string  `json:"projectLead" unstruct:"person"`
	TeamSize    int     `json:"teamSize" unstruct:"project"`
}

// BatchProcessingResult aggregates results from multiple documents
type BatchProcessingResult struct {
	TotalDocuments   int                      `json:"totalDocuments"`
	ProcessedFiles   []string                 `json:"processedFiles"`
	DocumentMetadata []DocumentMetadata       `json:"documentMetadata"`
	ProjectInfo      []ProjectInfo            `json:"projectInfo"`
	ProcessingStats  ProcessingStats          `json:"processingStats"`
	FileMetadata     []*unstruct.FileMetadata `json:"fileMetadata,omitempty"`
}

// ProcessingStats tracks batch processing performance
type ProcessingStats struct {
	StartTime   time.Time     `json:"startTime"`
	EndTime     time.Time     `json:"endTime"`
	Duration    time.Duration `json:"duration"`
	TotalSize   int64         `json:"totalSize"`
	AverageSize int64         `json:"averageSize"`
	SuccessRate float64       `json:"successRate"`
}

func main() {
	ctx := context.Background()

	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  GEMINI_API_KEY environment variable not set")
		fmt.Println("🔄 Running in demo mode with mock data...")
		fmt.Println("")
		fmt.Println("To run with real API calls, set your API key:")
		fmt.Println("export GEMINI_API_KEY=\"your-actual-api-key\"")
		fmt.Println("")
		runDemoMode()
		return
	}

	// Create Google GenAI client
	fmt.Println("Creating Google GenAI client...")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Create Stick-based prompt provider from template files
	fmt.Println("Setting up Stick template engine...")
	promptProvider, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
	)
	if err != nil {
		log.Fatal("Failed to create Stick prompt provider:", err)
	}

	// Example 1: Basic batch processing with progress tracking
	fmt.Println("\n=== Basic Batch Processing Example ===")
	runBasicBatchExample(ctx, client, promptProvider)

	// Example 2: Advanced batch processing with metadata and cleanup
	fmt.Println("\n=== Advanced Batch Processing Example ===")
	runAdvancedBatchExample(ctx, client, promptProvider)

	// Example 3: Mixed document types batch processing
	fmt.Println("\n=== Mixed Document Types Batch Example ===")
	runMixedBatchExample(ctx, client, promptProvider)

	// Example 4: Batch dry run for cost estimation
	fmt.Println("\n=== Batch Dry Run Example ===")
	runBatchDryRunExample(ctx, client, promptProvider)
}

func runBasicBatchExample(ctx context.Context, client *genai.Client, promptProvider unstruct.PromptProvider) {
	// Find all markdown files in the docs directory
	markdownFiles := findMarkdownFiles("docs")
	if len(markdownFiles) == 0 {
		fmt.Println("No markdown files found in docs/ directory")
		createSampleDocuments()
		markdownFiles = findMarkdownFiles("docs")
	}

	if len(markdownFiles) == 0 {
		fmt.Println("Still no files found, skipping basic batch example")
		return
	}

	fmt.Printf("Found %d markdown files to process\n", len(markdownFiles))

	// Create progress callback
	progressCallback := func(processed, total int, currentFile string) {
		if currentFile != "" {
			fmt.Printf("Processing file %d/%d: %s\n", processed+1, total, filepath.Base(currentFile))
		} else {
			fmt.Printf("Batch processing complete: %d/%d files processed\n", processed, total)
		}
	}

	// Create BatchFileAsset with progress tracking
	batchAsset := unstruct.NewBatchFileAsset(
		client,
		markdownFiles,
		unstruct.WithBatchProgressCallback(progressCallback),
	)

	// Create unstructor for document metadata
	u := unstruct.New[DocumentMetadata](client, promptProvider)

	assets := []unstruct.Asset{batchAsset}

	fmt.Println("Starting batch processing...")
	startTime := time.Now()

	result, err := u.Unstruct(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error in batch processing: %v", err)
		fmt.Println("\n❌ Batch processing failed")
		fmt.Println("💡 This might be due to:")
		fmt.Println("   - Invalid or expired API key")
		fmt.Println("   - Network connectivity issues") 
		fmt.Println("   - API rate limits")
		fmt.Println("   - File upload failures")
		fmt.Println("\n🔧 Try running without GEMINI_API_KEY set to see demo mode")
		return
	}

	duration := time.Since(startTime)
	fmt.Printf("\nBatch processing completed in %v\n", duration)

	// Display results
	fmt.Printf("Extracted Document Metadata:\n")
	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Description: %s\n", result.Description)
	fmt.Printf("Category: %s\n", result.Category)
	fmt.Printf("Author: %s\n", result.Author)
	fmt.Printf("Date: %s\n", result.Date)
	fmt.Printf("Version: %s\n", result.Version)
}

func runAdvancedBatchExample(ctx context.Context, client *genai.Client, promptProvider unstruct.PromptProvider) {
	// Find markdown files for advanced processing
	markdownFiles := findMarkdownFiles("docs")
	if len(markdownFiles) == 0 {
		fmt.Println("No markdown files found for advanced batch processing")
		return
	}

	fmt.Printf("Processing %d files with advanced options\n", len(markdownFiles))

	// Advanced progress callback with more details
	var processedFiles []string
	var totalSize int64

	progressCallback := func(processed, total int, currentFile string) {
		if currentFile != "" {
			if info, err := os.Stat(currentFile); err == nil {
				totalSize += info.Size()
				processedFiles = append(processedFiles, currentFile)
			}
			fmt.Printf("[%s] Processing %d/%d: %s (%.2f KB)\n",
				time.Now().Format("15:04:05"), processed+1, total,
				filepath.Base(currentFile), float64(totalSize)/1024)
		} else {
			fmt.Printf("[%s] ✅ Batch complete: %d files, %.2f KB total\n",
				time.Now().Format("15:04:05"), processed, float64(totalSize)/1024)
		}
	}

	// Create BatchFileAsset with advanced features
	batchAsset := unstruct.NewBatchFileAsset(
		client,
		markdownFiles,
		unstruct.WithBatchProgressCallback(progressCallback),
		unstruct.WithBatchIncludeMetadata(true),
		unstruct.WithBatchAutoCleanup(true),
		unstruct.WithBatchRetentionDays(7),
	)

	// Create unstructor for project information
	u := unstruct.New[ProjectInfo](client, promptProvider)

	assets := []unstruct.Asset{batchAsset}

	fmt.Println("Starting advanced batch processing...")
	startTime := time.Now()

	result, err := u.Unstruct(ctx, assets, unstruct.WithModel("gemini-1.5-pro"))
	if err != nil {
		log.Printf("Error in advanced batch processing: %v", err)
		fmt.Println("\n❌ Advanced batch processing failed")
		fmt.Println("💡 This might be due to:")
		fmt.Println("   - Invalid or expired API key")
		fmt.Println("   - Network connectivity issues") 
		fmt.Println("   - API rate limits")
		fmt.Println("   - File upload failures")
		return
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Display comprehensive results
	fmt.Printf("\n=== Advanced Batch Results ===\n")
	fmt.Printf("Project Code: %s\n", result.ProjectCode)
	fmt.Printf("Project Name: %s\n", result.ProjectName)
	fmt.Printf("Budget: %.2f %s\n", result.Budget, result.Currency)
	fmt.Printf("Timeline: %s to %s\n", result.StartDate, result.EndDate)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Priority: %s\n", result.Priority)
	fmt.Printf("Project Lead: %s\n", result.ProjectLead)
	fmt.Printf("Team Size: %d\n", result.TeamSize)

	// Display processing statistics
	fmt.Printf("\n=== Processing Statistics ===\n")
	fmt.Printf("Total files processed: %d\n", len(processedFiles))
	fmt.Printf("Total processing time: %v\n", duration)
	fmt.Printf("Average time per file: %v\n", duration/time.Duration(len(processedFiles)))
	fmt.Printf("Total data processed: %.2f KB\n", float64(totalSize)/1024)
	fmt.Printf("Processing throughput: %.2f KB/sec\n", float64(totalSize)/duration.Seconds()/1024)

	// Cleanup uploaded files (if auto cleanup is enabled)
	if err := batchAsset.Cleanup(ctx); err != nil {
		log.Printf("Warning: failed to cleanup batch files: %v", err)
	} else {
		fmt.Println("✅ Batch cleanup completed")
	}
}

func runMixedBatchExample(ctx context.Context, client *genai.Client, promptProvider unstruct.PromptProvider) {
	// Find different types of files for mixed processing
	allFiles := []string{}

	// Add markdown files
	markdownFiles := findMarkdownFiles("docs")
	allFiles = append(allFiles, markdownFiles...)

	// Add any text files or other document types
	textFiles := findTextFiles("docs")
	allFiles = append(allFiles, textFiles...)

	if len(allFiles) == 0 {
		fmt.Println("No files found for mixed batch processing")
		return
	}

	fmt.Printf("Processing %d mixed document types\n", len(allFiles))

	// Mixed processing callback
	progressCallback := func(processed, total int, currentFile string) {
		if currentFile != "" {
			ext := strings.ToLower(filepath.Ext(currentFile))
			fmt.Printf("📄 [%s] Processing %d/%d: %s\n",
				ext, processed+1, total, filepath.Base(currentFile))
		} else {
			fmt.Printf("🎯 Mixed batch processing complete: %d/%d files\n", processed, total)
		}
	}

	// Create BatchFileAsset for mixed content
	batchAsset := unstruct.NewBatchFileAsset(
		client,
		allFiles,
		unstruct.WithBatchProgressCallback(progressCallback),
		unstruct.WithBatchIncludeMetadata(true),
	)

	// Process with document metadata extraction
	u := unstruct.New[DocumentMetadata](client, promptProvider)
	assets := []unstruct.Asset{batchAsset}

	result, err := u.Unstruct(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error in mixed batch processing: %v", err)
		fmt.Println("\n❌ Mixed batch processing failed")
		fmt.Println("💡 This might be due to:")
		fmt.Println("   - Invalid or expired API key")
		fmt.Println("   - Network connectivity issues") 
		fmt.Println("   - Different file formats causing issues")
		fmt.Println("   - File upload failures")
		return
	}

	// Display results
	fmt.Printf("\n=== Mixed Batch Results ===\n")
	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Description: %s\n", result.Description)
	fmt.Printf("Category: %s\n", result.Category)
	fmt.Printf("Author: %s\n", result.Author)
	fmt.Printf("Date: %s\n", result.Date)
	fmt.Printf("Version: %s\n", result.Version)
}

func runBatchDryRunExample(ctx context.Context, client *genai.Client, promptProvider unstruct.PromptProvider) {
	// Find files for cost estimation
	markdownFiles := findMarkdownFiles("docs")
	if len(markdownFiles) == 0 {
		fmt.Println("No files found for batch dry run")
		return
	}

	fmt.Printf("Estimating costs for batch processing %d files\n", len(markdownFiles))

	// Create BatchFileAsset for dry run
	batchAsset := unstruct.NewBatchFileAsset(client, markdownFiles)

	// Create unstructor for cost estimation
	u := unstruct.New[DocumentMetadata](client, promptProvider)
	assets := []unstruct.Asset{batchAsset}

	stats, err := u.DryRun(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error in batch dry run: %v", err)
		return
	}

	fmt.Printf("\n=== Batch Cost Estimation ===\n")
	fmt.Printf("Total files: %d\n", len(markdownFiles))
	fmt.Printf("Estimated prompt calls: %d\n", stats.PromptCalls)
	fmt.Printf("Estimated input tokens: %d\n", stats.TotalInputTokens)
	fmt.Printf("Estimated output tokens: %d\n", stats.TotalOutputTokens)
	fmt.Printf("Average tokens per file: %.1f input, %.1f output\n",
		float64(stats.TotalInputTokens)/float64(len(markdownFiles)),
		float64(stats.TotalOutputTokens)/float64(len(markdownFiles)))
	
	// Calculate estimated costs (Gemini 1.5 Flash pricing)
	inputCostPer1M := 0.075  // $0.075 per 1M input tokens
	outputCostPer1M := 0.30  // $0.30 per 1M output tokens
	
	inputCost := float64(stats.TotalInputTokens) / 1000000 * inputCostPer1M
	outputCost := float64(stats.TotalOutputTokens) / 1000000 * outputCostPer1M
	totalCost := inputCost + outputCost
	
	fmt.Printf("\n=== Estimated Costs (Gemini 1.5 Flash) ===\n")
	fmt.Printf("Input tokens cost: $%.6f (%.3f M tokens @ $%.3f/M)\n", inputCost, float64(stats.TotalInputTokens)/1000000, inputCostPer1M)
	fmt.Printf("Output tokens cost: $%.6f (%.3f M tokens @ $%.3f/M)\n", outputCost, float64(stats.TotalOutputTokens)/1000000, outputCostPer1M)
	fmt.Printf("Total estimated cost: $%.6f\n", totalCost)
	fmt.Printf("Average cost per file: $%.6f\n", totalCost/float64(len(markdownFiles)))
	
	if len(markdownFiles) > 1 {
		fmt.Printf("\n=== Batch Processing Benefits ===\n")
		fmt.Printf("Files processed in batch: %d\n", len(markdownFiles))
		fmt.Printf("Prompt calls saved: %d (vs %d individual calls)\n", len(markdownFiles)-stats.PromptCalls, len(markdownFiles))
		if len(markdownFiles) > stats.PromptCalls {
			efficiency := float64(len(markdownFiles)-stats.PromptCalls) / float64(len(markdownFiles)) * 100
			fmt.Printf("Processing efficiency: %.1f%% reduction in API calls\n", efficiency)
		}
	}
}

// findMarkdownFiles looks for .md files in the specified directory
func findMarkdownFiles(dir string) []string {
	return findFilesByExtension(dir, ".md")
}

// findTextFiles looks for .txt files in the specified directory
func findTextFiles(dir string) []string {
	return findFilesByExtension(dir, ".txt")
}

// findFilesByExtension looks for files with a specific extension
func findFilesByExtension(dir, ext string) []string {
	var files []string

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return files
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.EqualFold(filepath.Ext(entry.Name()), ext) {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files
}

// createSampleDocuments creates sample markdown files for testing
func createSampleDocuments() {
	fmt.Println("Creating sample documents for batch processing...")

	// Create docs directory if it doesn't exist
	if err := os.MkdirAll("docs", 0755); err != nil {
		log.Printf("Failed to create docs directory: %v", err)
		return
	}

	// Sample documents
	documents := map[string]string{
		"project-alpha.md": `# Project Alpha - Development Plan

**Author:** John Smith  
**Date:** January 15, 2024  
**Version:** 2.1  
**Category:** Software Development

## Project Overview

Project Alpha is a cutting-edge web application designed to streamline business processes.

**Project Details:**
- Project Code: PROJ-ALPHA-2024
- Project Name: Alpha Business Suite
- Budget: $150,000.00 USD
- Start Date: February 1, 2024
- End Date: August 31, 2024
- Status: In Progress
- Priority: High
- Project Lead: Jane Doe
- Team Size: 8 developers

## Technical Specifications

The application will be built using modern web technologies including React, Node.js, and PostgreSQL.
`,

		"meeting-notes.md": `# Weekly Team Meeting - March 5, 2024

**Author:** Sarah Johnson  
**Date:** March 5, 2024  
**Version:** 1.0  
**Category:** Meeting Minutes

## Project Beta Updates

**Project Information:**
- Project Code: PROJ-BETA-2024
- Project Name: Beta Analytics Platform
- Budget: $220,000.00 USD
- Start Date: March 1, 2024
- End Date: November 30, 2024
- Status: Planning
- Priority: Medium
- Project Lead: Michael Chen
- Team Size: 12 developers

## Discussion Points

1. Technical architecture review
2. Budget allocation for Q2
3. Resource planning for the analytics platform
`,

		"research-doc.md": `# Machine Learning Research Document

**Author:** Dr. Emily Watson  
**Date:** April 20, 2024  
**Version:** 3.2  
**Category:** Research

## Abstract

This document outlines the research findings for our machine learning initiatives.

**Project Gamma Details:**
- Project Code: PROJ-GAMMA-2024
- Project Name: Gamma ML Framework
- Budget: $300,000.00 USD
- Start Date: May 1, 2024
- End Date: December 31, 2024
- Status: Research Phase
- Priority: High
- Project Lead: Dr. Robert Kim
- Team Size: 6 researchers

## Methodology

The research focuses on developing advanced neural network architectures for natural language processing.
`,
	}

	for filename, content := range documents {
		filepath := filepath.Join("docs", filename)
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			log.Printf("Failed to create sample document %s: %v", filename, err)
		}
	}

	fmt.Printf("Created %d sample documents in docs/ directory\n", len(documents))
}

// runDemoMode shows example output when API key is not available
func runDemoMode() {
	// Find documents to show file count
	markdownFiles := findMarkdownFiles("docs")
	if len(markdownFiles) == 0 {
		fmt.Println("Creating sample documents for demo...")
		createSampleDocuments()
		markdownFiles = findMarkdownFiles("docs")
	}

	fmt.Printf("Found %d sample documents for demonstration\n", len(markdownFiles))
	
	// Show file list
	fmt.Println("\n=== Sample Documents ===")
	for i, file := range markdownFiles {
		if info, err := os.Stat(file); err == nil {
			fmt.Printf("%d. %s (%.1f KB)\n", i+1, filepath.Base(file), float64(info.Size())/1024)
		}
	}

	// Demo batch processing output
	fmt.Println("\n=== Demo: Basic Batch Processing ===")
	for i, file := range markdownFiles {
		fmt.Printf("Processing file %d/%d: %s\n", i+1, len(markdownFiles), filepath.Base(file))
		time.Sleep(100 * time.Millisecond) // Simulate processing
	}
	fmt.Printf("Batch processing complete: %d/%d files processed\n", len(markdownFiles), len(markdownFiles))
	fmt.Println("\nDemo Extracted Document Metadata:")
	fmt.Println("Title: Project Alpha - Development Plan")
	fmt.Println("Description: A cutting-edge web application designed to streamline business processes")
	fmt.Println("Category: Software Development")
	fmt.Println("Author: John Smith")
	fmt.Println("Date: January 15, 2024")
	fmt.Println("Version: 2.1")

	// Demo advanced processing
	fmt.Println("\n=== Demo: Advanced Batch Processing ===")
	totalSize := int64(0)
	for i, file := range markdownFiles {
		if info, err := os.Stat(file); err == nil {
			totalSize += info.Size()
		}
		fmt.Printf("[%s] Processing %d/%d: %s (%.2f KB)\n", 
			time.Now().Format("15:04:05"), i+1, len(markdownFiles), 
			filepath.Base(file), float64(totalSize)/1024)
		time.Sleep(150 * time.Millisecond) // Simulate processing
	}
	fmt.Printf("[%s] ✅ Batch complete: %d files, %.2f KB total\n", 
		time.Now().Format("15:04:05"), len(markdownFiles), float64(totalSize)/1024)

	fmt.Println("\nDemo Advanced Batch Results:")
	fmt.Println("Project Code: PROJ-ALPHA-2024")
	fmt.Println("Project Name: Alpha Business Suite")
	fmt.Println("Budget: 150000.00 USD")
	fmt.Println("Timeline: February 1, 2024 to August 31, 2024")
	fmt.Println("Status: In Progress")
	fmt.Println("Priority: High")
	fmt.Println("Project Lead: Jane Doe")
	fmt.Println("Team Size: 8")

	fmt.Println("\n=== Demo: Processing Statistics ===")
	fmt.Printf("Total files processed: %d\n", len(markdownFiles))
	fmt.Println("Total processing time: 2.34s")
	fmt.Println("Average time per file: 390ms")
	fmt.Printf("Total data processed: %.2f KB\n", float64(totalSize)/1024)
	fmt.Printf("Processing throughput: %.2f KB/sec\n", float64(totalSize)/1024/2.34)
	fmt.Println("✅ Demo batch cleanup completed")

	// Demo cost estimation
	fmt.Println("\n=== Demo: Batch Cost Estimation ===")
	fmt.Printf("Total files: %d\n", len(markdownFiles))
	fmt.Println("Estimated prompt calls: 2")
	fmt.Println("Estimated input tokens: 15,840")
	fmt.Println("Estimated output tokens: 850")
	fmt.Printf("Average tokens per file: %.1f input, %.1f output\n", 15840.0/float64(len(markdownFiles)), 850.0/float64(len(markdownFiles)))

	// Demo cost calculation
	inputCost := 15840.0 / 1000000 * 0.075
	outputCost := 850.0 / 1000000 * 0.30
	totalCost := inputCost + outputCost
	
	fmt.Println("\n=== Demo: Estimated Costs (Gemini 1.5 Flash) ===")
	fmt.Printf("Input tokens cost: $%.6f (%.3f M tokens @ $%.3f/M)\n", inputCost, 15840.0/1000000, 0.075)
	fmt.Printf("Output tokens cost: $%.6f (%.3f M tokens @ $%.3f/M)\n", outputCost, 850.0/1000000, 0.30)
	fmt.Printf("Total estimated cost: $%.6f\n", totalCost)
	fmt.Printf("Average cost per file: $%.6f\n", totalCost/float64(len(markdownFiles)))
	
	if len(markdownFiles) > 2 {
		fmt.Println("\n=== Demo: Batch Processing Benefits ===")
		fmt.Printf("Files processed in batch: %d\n", len(markdownFiles))
		fmt.Printf("Prompt calls saved: %d (vs %d individual calls)\n", len(markdownFiles)-2, len(markdownFiles))
		efficiency := float64(len(markdownFiles)-2) / float64(len(markdownFiles)) * 100
		fmt.Printf("Processing efficiency: %.1f%% reduction in API calls\n", efficiency)
	}

	fmt.Println("\n🚀 Demo complete! Set GEMINI_API_KEY to run with real API calls.")
}
