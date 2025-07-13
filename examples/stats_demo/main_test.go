package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	unstruct "github.com/vivaneiona/genkit-unstruct"
)

func TestSimplePromptProvider(t *testing.T) {
	provider := NewSimplePromptProvider()
	
	// Test existing template
	template, err := provider.GetPrompt("person-info", 1)
	if err != nil {
		t.Fatalf("Expected template to exist, got error: %v", err)
	}
	
	if !strings.Contains(template, "Extract personal information") {
		t.Errorf("Template should contain expected content")
	}
	
	// Test non-existing template
	_, err = provider.GetPrompt("non-existent", 1)
	if err == nil {
		t.Error("Expected error for non-existent template")
	}
}

func TestExecutionStatsStructure(t *testing.T) {
	stats := &unstruct.ExecutionStats{
		PromptCalls:       2,
		ModelCalls:        map[string]int{"gpt-4o": 1, "gpt-3.5-turbo": 1},
		PromptGroups:      2,
		FieldsExtracted:   4,
		TotalInputTokens:  250,
		TotalOutputTokens: 80,
		GroupDetails: []unstruct.GroupExecution{
			{
				PromptName:   "test-prompt",
				Model:        "test-model",
				Fields:       []string{"field1", "field2"},
				InputTokens:  100,
				OutputTokens: 50,
			},
		},
	}
	
	if stats.PromptCalls != 2 {
		t.Errorf("Expected 2 prompt calls, got %d", stats.PromptCalls)
	}
	
	if len(stats.ModelCalls) != 2 {
		t.Errorf("Expected 2 model entries, got %d", len(stats.ModelCalls))
	}
	
	if len(stats.GroupDetails) != 1 {
		t.Errorf("Expected 1 group detail, got %d", len(stats.GroupDetails))
	}
}

func TestDemoWithoutAPI(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	sampleDoc := "Test document"
	demoWithoutAPI(sampleDoc)
	
	// Restore stdout
	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	
	// Check that demo output contains expected elements
	if !strings.Contains(output, "Demo Statistics") {
		t.Error("Demo output should contain 'Demo Statistics'")
	}
	
	if !strings.Contains(output, "person-info") {
		t.Error("Demo output should contain prompt names")
	}
	
	if !strings.Contains(output, "Token Analysis") {
		t.Error("Demo output should contain token analysis")
	}
}

func TestTestDocumentStructure(t *testing.T) {
	// Test that our test document structure is correctly defined
	doc := TestDocument{
		Name:    "John Doe",
		Age:     35,
		Email:   "john@example.com",
		Address: "123 Main St",
	}
	
	if doc.Name == "" {
		t.Error("Name field should be set")
	}
	
	if doc.Age == 0 {
		t.Error("Age field should be set")
	}
	
	if doc.Email == "" {
		t.Error("Email field should be set")
	}
	
	if doc.Address == "" {
		t.Error("Address field should be set")
	}
}

func TestPrintFunctions(t *testing.T) {
	// Test that print functions don't panic
	stats := &unstruct.ExecutionStats{
		PromptCalls:       1,
		ModelCalls:        map[string]int{"test-model": 1},
		PromptGroups:      1,
		FieldsExtracted:   2,
		TotalInputTokens:  100,
		TotalOutputTokens: 50,
		GroupDetails: []unstruct.GroupExecution{
			{
				PromptName:   "test-prompt",
				Model:        "test-model",
				Fields:       []string{"field1", "field2"},
				InputTokens:  100,
				OutputTokens: 50,
			},
		},
	}
	
	// Capture stdout to avoid output during testing
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	
	// Test functions don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Print function panicked: %v", r)
		}
	}()
	
	printExecutionStats(stats)
	printTokenAnalysis(stats)
	printSectionHeader("Test Section")
	
	// Restore stdout
	w.Close()
	os.Stdout = old
}

// Integration test that tests the main demo flow without API
func TestMainDemoFlow(t *testing.T) {
	// Save original environment
	originalKey := os.Getenv("GEMINI_API_KEY")
	defer os.Setenv("GEMINI_API_KEY", originalKey)
	
	// Unset API key to trigger demo mode
	os.Unsetenv("GEMINI_API_KEY")
	
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	// Run main (this should trigger demo mode)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Main function panicked: %v", r)
			}
		}()
		main()
	}()
	
	// Restore stdout
	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	
	// Verify demo mode was triggered
	if !strings.Contains(output, "demo mode") {
		t.Error("Should have triggered demo mode")
	}
	
	// Verify key sections were printed
	expectedSections := []string{
		"Execution Statistics Collection Demo",
		"Demo Statistics", 
		"Token Analysis",
		"GEMINI_API_KEY",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Output should contain '%s'", section)
		}
	}
}

// Benchmark the stats generation
func BenchmarkStatsGeneration(b *testing.B) {
	stats := &unstruct.ExecutionStats{
		PromptCalls:       2,
		ModelCalls:        map[string]int{"gpt-4o": 1, "gpt-3.5-turbo": 1},
		PromptGroups:      2,
		FieldsExtracted:   4,
		TotalInputTokens:  250,
		TotalOutputTokens: 80,
		GroupDetails: []unstruct.GroupExecution{
			{
				PromptName:   "person-info",
				Model:        "gpt-4o",
				Fields:       []string{"name", "age"},
				InputTokens:  120,
				OutputTokens: 40,
			},
			{
				PromptName:   "contact-info",
				Model:        "gpt-3.5-turbo",
				Fields:       []string{"email", "address"},
				InputTokens:  130,
				OutputTokens: 40,
			},
		},
	}
	
	// Capture stdout to avoid output during benchmarking
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		printExecutionStats(stats)
		printTokenAnalysis(stats)
	}
	
	// Restore stdout
	w.Close()
	os.Stdout = old
}
