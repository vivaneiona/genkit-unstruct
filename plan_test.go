package unstruct

import (
	"testing"
)

func TestPlanBuilder_Explain(t *testing.T) {
	builder := NewPlanBuilder()

	// Set up a simple schema
	schema := map[string]interface{}{
		"fields": []string{"Name", "Age", "Address", "Email"},
	}

	builder.WithSchema(schema)

	// Generate explanation
	plan, err := builder.Explain()
	if err != nil {
		t.Fatalf("Failed to generate plan: %v", err)
	}

	// Verify root node
	if plan.Type != SchemaAnalysisType {
		t.Errorf("Expected root node type %s, got %s", SchemaAnalysisType, plan.Type)
	}

	if len(plan.Fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(plan.Fields))
	}

	// Should have 4 PromptCall children + 1 MergeFragments child
	if len(plan.Children) != 5 {
		t.Errorf("Expected 5 children, got %d", len(plan.Children))
	}

	// Verify PromptCall nodes
	promptCallCount := 0
	mergeFragmentCount := 0

	for _, child := range plan.Children {
		switch child.Type {
		case PromptCallType:
			promptCallCount++
			if child.Model == "" {
				t.Error("PromptCall node should have a model")
			}
			if child.PromptName == "" {
				t.Error("PromptCall node should have a prompt name")
			}
			if len(child.Fields) != 1 {
				t.Error("PromptCall node should have exactly one field")
			}
		case MergeFragmentsType:
			mergeFragmentCount++
		}
	}

	if promptCallCount != 4 {
		t.Errorf("Expected 4 PromptCall nodes, got %d", promptCallCount)
	}

	if mergeFragmentCount != 1 {
		t.Errorf("Expected 1 MergeFragments node, got %d", mergeFragmentCount)
	}

	// Verify costs are calculated
	if plan.EstCost <= 0 {
		t.Error("Root node should have positive estimated cost")
	}
}

func TestPlanBuilder_ExplainWithCosts(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"Name", "Age"},
	}

	builder.WithSchema(schema)

	pricing := DefaultModelPricing()

	plan, err := builder.ExplainWithCosts(pricing)
	if err != nil {
		t.Fatalf("Failed to generate plan with costs: %v", err)
	}

	// Verify actual costs are calculated for PromptCall nodes
	foundActualCost := false
	for _, child := range plan.Children {
		if child.Type == PromptCallType && child.ActCost != nil {
			foundActualCost = true
			if *child.ActCost <= 0 {
				t.Error("Actual cost should be positive")
			}
		}
	}

	if !foundActualCost {
		t.Error("Should have found at least one PromptCall with actual cost")
	}
}

func TestFormatAsText(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"Name", "Age"},
	}

	builder.WithSchema(schema)

	textOutput, err := builder.ExplainPretty(FormatText)
	if err != nil {
		t.Fatalf("Failed to format as text: %v", err)
	}

	// Verify text output contains expected elements
	expectedStrings := []string{
		"Unstructor Execution Plan",
		"SchemaAnalysis",
		"PromptCall",
		"MergeFragments",
		"NameExtractionPrompt",
		"AgeExtractionPrompt",
		"cost=",
		"tokens(in=",
	}

	for _, expected := range expectedStrings {
		if !contains(textOutput, expected) {
			t.Errorf("Text output should contain '%s'", expected)
		}
	}
}

func TestFormatAsJSON(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"Name"},
	}

	builder.WithSchema(schema)

	jsonOutput, err := builder.ExplainPretty(FormatJSON)
	if err != nil {
		t.Fatalf("Failed to format as JSON: %v", err)
	}

	// Verify JSON output is valid by checking for key elements
	expectedStrings := []string{
		`"type": "SchemaAnalysis"`,
		`"type": "PromptCall"`,
		`"type": "MergeFragments"`,
		`"promptName": "NameExtractionPrompt"`,
		`"model": "gpt-3.5-turbo"`,
		`"estCost"`,
		`"children"`,
	}

	for _, expected := range expectedStrings {
		if !contains(jsonOutput, expected) {
			t.Errorf("JSON output should contain '%s'", expected)
		}
	}
}

func TestFormatAsGraphviz(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"Name"},
	}

	builder.WithSchema(schema)

	dotOutput, err := builder.ExplainPretty(FormatGraphviz)
	if err != nil {
		t.Fatalf("Failed to format as Graphviz: %v", err)
	}

	// Verify DOT output contains expected elements
	expectedStrings := []string{
		"digraph UnstructorPlan",
		"rankdir=TB",
		"node0",
		"->",
		"SchemaAnalysis",
		"PromptCall",
		"MergeFragments",
	}

	for _, expected := range expectedStrings {
		if !contains(dotOutput, expected) {
			t.Errorf("DOT output should contain '%s'", expected)
		}
	}
}

func TestFormatAsHTML(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"Name"},
	}

	builder.WithSchema(schema)

	htmlOutput, err := builder.ExplainPretty(FormatHTML)
	if err != nil {
		t.Fatalf("Failed to format as HTML: %v", err)
	}

	// Verify HTML output contains expected elements
	expectedStrings := []string{
		"<!DOCTYPE html>",
		"<title>Unstructor Execution Plan</title>",
		"SchemaAnalysis",
		"PromptCall",
	}

	for _, expected := range expectedStrings {
		if !contains(htmlOutput, expected) {
			t.Errorf("HTML output should contain '%s'", expected)
		}
	}
}

func TestTokenEstimation(t *testing.T) {
	// Test text-based estimation
	text := "This is a sample text for token estimation."
	tokens := EstimateTokensFromText(text)
	if tokens <= 0 {
		t.Error("Token estimation should return positive value")
	}

	// Test word-based estimation
	wordCount := 10
	tokensFromWords := EstimateTokensFromWords(wordCount)
	if tokensFromWords <= 0 {
		t.Error("Word-based token estimation should return positive value")
	}
}

func TestDefaultModelPricing(t *testing.T) {
	pricing := DefaultModelPricing()

	expectedModels := []string{"gpt-4o", "gpt-3.5-turbo", "claude-3-sonnet"}

	for _, model := range expectedModels {
		if price, exists := pricing[model]; !exists {
			t.Errorf("Default pricing should include %s", model)
		} else {
			if price.PromptTokCost <= 0 || price.CompletionTokCost <= 0 {
				t.Errorf("Model %s should have positive pricing", model)
			}
		}
	}
}

func TestModelConfiguration(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"Name", "Age"},
	}

	modelConfig := map[string]string{
		"Name": "gpt-4",
		"Age":  "gpt-3.5-turbo",
	}

	builder.WithSchema(schema).WithModelConfig(modelConfig)

	plan, err := builder.Explain()
	if err != nil {
		t.Fatalf("Failed to generate plan: %v", err)
	}

	// Verify models are correctly assigned
	modelAssignments := make(map[string]string)
	for _, child := range plan.Children {
		if child.Type == PromptCallType && len(child.Fields) == 1 {
			modelAssignments[child.Fields[0]] = child.Model
		}
	}

	if modelAssignments["Name"] != "gpt-4" {
		t.Errorf("Expected Name field to use gpt-4, got %s", modelAssignments["Name"])
	}

	if modelAssignments["Age"] != "gpt-3.5-turbo" {
		t.Errorf("Expected Age field to use gpt-3.5-turbo, got %s", modelAssignments["Age"])
	}
}

func TestCostCalculation(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"Name", "Age"},
	}

	builder.WithSchema(schema)

	plan, err := builder.Explain()
	if err != nil {
		t.Fatalf("Failed to generate plan: %v", err)
	}

	// Verify cost hierarchy (parent cost should include children)
	childrenTotalCost := 0.0
	for _, child := range plan.Children {
		childrenTotalCost += child.EstCost
	}

	// Root cost should be at least the sum of children costs
	if plan.EstCost < childrenTotalCost {
		t.Errorf("Root cost (%f) should be at least sum of children costs (%f)", plan.EstCost, childrenTotalCost)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
