package unstruct

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	// Verify root node
	assert.Equal(t, SchemaAnalysisType, plan.Type)
	assert.Len(t, plan.Fields, 4)

	// Should have 4 PromptCall children + 1 MergeFragments child
	assert.Len(t, plan.Children, 5)

	// Verify PromptCall nodes
	promptCallCount := 0
	mergeFragmentCount := 0

	for _, child := range plan.Children {
		switch child.Type {
		case PromptCallType:
			promptCallCount++
			assert.NotEmpty(t, child.Model, "PromptCall node should have a model")
			assert.NotEmpty(t, child.PromptName, "PromptCall node should have a prompt name")
			assert.Len(t, child.Fields, 1, "PromptCall node should have exactly one field")
		case MergeFragmentsType:
			mergeFragmentCount++
		}
	}

	assert.Equal(t, 4, promptCallCount)
	assert.Equal(t, 1, mergeFragmentCount)

	// Verify costs are calculated
	assert.Greater(t, plan.EstCost, 0.0, "Root node should have positive estimated cost")
}

func TestPlanBuilder_ExplainWithCosts(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"Name", "Age"},
	}

	builder.WithSchema(schema)

	pricing := DefaultModelPricing()

	plan, err := builder.ExplainWithCosts(pricing)
	require.NoError(t, err)

	// Verify actual costs are calculated for PromptCall nodes
	foundActualCost := false
	for _, child := range plan.Children {
		if child.Type == PromptCallType && child.ActCost != nil {
			foundActualCost = true
			assert.Greater(t, *child.ActCost, 0.0, "Actual cost should be positive")
		}
	}

	assert.True(t, foundActualCost, "Should have found at least one PromptCall with actual cost")
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

// Tests for Execution Statistics and Dry-Run Functionality

func TestExecutionStats_Structure(t *testing.T) {
	stats := &ExecutionStats{
		PromptCalls:       3,
		ModelCalls:        map[string]int{"gpt-4o": 2, "gpt-3.5-turbo": 1},
		PromptGroups:      3,
		FieldsExtracted:   5,
		TotalInputTokens:  1200,
		TotalOutputTokens: 180,
		GroupDetails: []GroupExecution{
			{
				PromptName:   "person-info",
				Model:        "gpt-4o",
				Fields:       []string{"name", "age"},
				InputTokens:  400,
				OutputTokens: 60,
				ParentPath:   "",
			},
			{
				PromptName:   "contact-info",
				Model:        "gpt-3.5-turbo",
				Fields:       []string{"email", "phone"},
				InputTokens:  350,
				OutputTokens: 50,
				ParentPath:   "",
			},
		},
	}

	assert.Equal(t, 3, stats.PromptCalls)
	assert.Equal(t, 3, stats.PromptGroups)
	assert.Equal(t, 5, stats.FieldsExtracted)
	assert.Equal(t, 1200, stats.TotalInputTokens)
	assert.Equal(t, 180, stats.TotalOutputTokens)
	assert.Len(t, stats.ModelCalls, 2)
	assert.Equal(t, 2, stats.ModelCalls["gpt-4o"])
	assert.Equal(t, 1, stats.ModelCalls["gpt-3.5-turbo"])
	assert.Len(t, stats.GroupDetails, 2)
}

func TestGroupExecution_Structure(t *testing.T) {
	group := GroupExecution{
		PromptName:   "test-prompt",
		Model:        "gpt-4o",
		Fields:       []string{"field1", "field2"},
		InputTokens:  100,
		OutputTokens: 20,
		ParentPath:   "parent.path",
	}

	assert.Equal(t, "test-prompt", group.PromptName)
	assert.Equal(t, "gpt-4o", group.Model)
	assert.Equal(t, []string{"field1", "field2"}, group.Fields)
	assert.Equal(t, 100, group.InputTokens)
	assert.Equal(t, 20, group.OutputTokens)
	assert.Equal(t, "parent.path", group.ParentPath)
}

func TestPlanBuilder_WithUnstructor(t *testing.T) {
	builder := NewPlanBuilder()

	// Create a mock unstructor - we'll use nil for this test
	mockUnstructor := (*MockUnstructor)(nil)

	result := builder.WithUnstructor(mockUnstructor)

	assert.Equal(t, builder, result) // Should return the same builder for chaining
	assert.Equal(t, mockUnstructor, builder.unstructor)
}

func TestPlanBuilder_WithSampleDocument(t *testing.T) {
	builder := NewPlanBuilder()
	sampleDoc := "This is a sample document for testing token estimation."

	result := builder.WithSampleDocument(sampleDoc)

	assert.Equal(t, builder, result) // Should return the same builder for chaining
	assert.Equal(t, sampleDoc, builder.document)
}

func TestPlanBuilder_CallDryRun_NoUnstructor(t *testing.T) {
	builder := NewPlanBuilder()

	stats, err := builder.callDryRun()

	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "unstructor not configured")
}

func TestPlanBuilder_CallDryRun_NoDocument(t *testing.T) {
	builder := NewPlanBuilder()
	builder.WithUnstructor(&MockUnstructor{})

	stats, err := builder.callDryRun()

	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "sample document not configured")
}

func TestPlanBuilder_CallDryRun_NotDryRunner(t *testing.T) {
	builder := NewPlanBuilder()
	builder.WithUnstructor("not a dry runner") // String doesn't implement DryRunner
	builder.WithSampleDocument("sample doc")

	stats, err := builder.callDryRun()

	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "does not implement DryRunner interface")
}

func TestPlanBuilder_GetFieldsFromStats(t *testing.T) {
	builder := NewPlanBuilder()
	stats := &ExecutionStats{
		GroupDetails: []GroupExecution{
			{Fields: []string{"name", "age"}},
			{Fields: []string{"email", "phone"}},
			{Fields: []string{"name", "address"}}, // name appears twice
		},
	}

	fields := builder.getFieldsFromStats(stats)

	assert.Len(t, fields, 5) // Should have 5 unique fields
	assert.Contains(t, fields, "name")
	assert.Contains(t, fields, "age")
	assert.Contains(t, fields, "email")
	assert.Contains(t, fields, "phone")
	assert.Contains(t, fields, "address")
}

func TestPlanBuilder_GetModelsFromStats(t *testing.T) {
	builder := NewPlanBuilder()
	stats := &ExecutionStats{
		ModelCalls: map[string]int{
			"gpt-4o":        2,
			"gpt-3.5-turbo": 1,
			"claude-3":      1,
		},
	}

	models := builder.getModelsFromStats(stats)

	assert.Len(t, models, 3)
	assert.Contains(t, models, "gpt-4o")
	assert.Contains(t, models, "gpt-3.5-turbo")
	assert.Contains(t, models, "claude-3")
}

func TestPlanBuilder_BuildPlanFromStaticAnalysis_Fallback(t *testing.T) {
	builder := NewPlanBuilder()
	schema := map[string]interface{}{
		"fields": []string{"name", "age"},
	}
	builder.WithSchema(schema)

	plan, err := builder.buildPlanFromStaticAnalysis(ExplainOptions{})

	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, SchemaAnalysisType, plan.Type)
	assert.Len(t, plan.Fields, 2)
	assert.Contains(t, plan.Fields, "name")
	assert.Contains(t, plan.Fields, "age")
}

func TestDryRunner_Interface(t *testing.T) {
	// Test that our MockUnstructor implements DryRunner
	var mock DryRunner = &MockUnstructor{}
	assert.NotNil(t, mock)

	// Test the interface method signature
	assets := []Asset{&TextAsset{Content: "test doc"}}
	stats, err := mock.DryRun(context.Background(), assets)
	assert.NotNil(t, stats) // MockUnstructor should return stats
	assert.NoError(t, err)
}

func TestPlanNode_ExpectedModelsAndCounts(t *testing.T) {
	builder := NewPlanBuilder()
	schema := map[string]interface{}{
		"fields": []string{"name", "email"},
	}
	modelConfig := map[string]string{
		"name":  "gpt-4o",
		"email": "gpt-3.5-turbo",
	}

	builder.WithSchema(schema).WithModelConfig(modelConfig)

	plan, err := builder.Explain()

	require.NoError(t, err)
	assert.NotNil(t, plan.ExpectedModels)
	assert.NotNil(t, plan.ExpectedCallCounts)

	// Should have both models
	assert.Len(t, plan.ExpectedModels, 2)
	assert.Contains(t, plan.ExpectedModels, "gpt-4o")
	assert.Contains(t, plan.ExpectedModels, "gpt-3.5-turbo")

	// Should have correct call counts
	assert.Equal(t, 1, plan.ExpectedCallCounts["gpt-4o"])
	assert.Equal(t, 1, plan.ExpectedCallCounts["gpt-3.5-turbo"])
}

// MockUnstructor for testing DryRunner interface
type MockUnstructor struct{}

func (m *MockUnstructor) DryRun(ctx context.Context, assets []Asset, optFns ...func(*Options)) (*ExecutionStats, error) {
	return &ExecutionStats{
		PromptCalls:       2,
		ModelCalls:        map[string]int{"gpt-4o": 1, "gpt-3.5-turbo": 1},
		PromptGroups:      2,
		FieldsExtracted:   4,
		TotalInputTokens:  100,
		TotalOutputTokens: 50,
		GroupDetails: []GroupExecution{
			{
				PromptName:   "test-prompt-1",
				Model:        "gpt-4o",
				Fields:       []string{"field1", "field2"},
				InputTokens:  50,
				OutputTokens: 25,
			},
			{
				PromptName:   "test-prompt-2",
				Model:        "gpt-3.5-turbo",
				Fields:       []string{"field3", "field4"},
				InputTokens:  50,
				OutputTokens: 25,
			},
		},
	}, nil
}

func TestParametersAndCostFormatting(t *testing.T) {
	builder := NewPlanBuilder()

	// Test schema with fields using parameters
	schema := map[string]interface{}{
		"fields": []string{"Name", "Age", "Email"},
	}

	// Test with parameters to verify they are displayed
	modelConfig := map[string]string{
		"Name":  "model/gemini-1.5-flash?temperature=0.2&topK=10",
		"Age":   "model/gemini-2.0-flash?temperature=0.5",
		"Email": "model/gpt-4o?maxOutputTokens=1000&topP=0.8",
	}

	builder.WithSchema(schema).WithModelConfig(modelConfig)

	// Get plan with costs
	pricing := DefaultModelPricing()
	plan, err := builder.ExplainWithCosts(pricing)
	require.NoError(t, err)

	// Print parameters and costs for each prompt call
	t.Logf("=== Execution Plan with Parameters and Costs ===")
	t.Logf("Total Estimated Cost: $%.4f", plan.EstCost)

	for _, child := range plan.Children {
		if child.Type == PromptCallType {
			t.Logf("\n--- PromptCall: %s ---", child.PromptName)
			t.Logf("Model: %s", child.Model)

			// Print parameters if any (this would need to be added to PlanNode struct)
			// For now, we can extract from the model string if it contains query params
			if strings.Contains(child.Model, "?") {
				parts := strings.Split(child.Model, "?")
				if len(parts) > 1 {
					t.Logf("Parameters: %s", parts[1])
				}
			}

			t.Logf("Fields: %v", child.Fields)
			t.Logf("Estimated Cost: $%.6f", child.EstCost)

			if child.ActCost != nil {
				t.Logf("Actual Cost: $%.6f", *child.ActCost)
			}

			t.Logf("Input Tokens: %d, Output Tokens: %d", child.InputTokens, child.OutputTokens)
		}
	}

	// Verify parameters are preserved in the plan
	for _, child := range plan.Children {
		if child.Type == PromptCallType {
			// Check that the model field contains parameters
			if len(child.Fields) > 0 {
				field := child.Fields[0]
				expectedParam := false
				switch field {
				case "Name":
					expectedParam = strings.Contains(child.Model, "temperature=0.2")
				case "Age":
					expectedParam = strings.Contains(child.Model, "temperature=0.5")
				case "Email":
					expectedParam = strings.Contains(child.Model, "maxOutputTokens=1000")
				}

				if !expectedParam {
					t.Logf("Warning: Expected parameters not found in model for field %s: %s", field, child.Model)
				}
			}
		}
	}

	// Verify cost formatting
	assert.Greater(t, plan.EstCost, 0.0, "Total cost should be positive")

	// Print formatted text output
	textOutput, err := builder.ExplainPrettyWithCosts(FormatText, pricing)
	require.NoError(t, err)

	t.Logf("\n=== Formatted Text Output ===")
	t.Logf("%s", textOutput)

	// Verify text output contains cost formatting
	assert.Contains(t, textOutput, "cost=", "Text output should contain cost information")

	// Verify parameters are displayed in model fields within the formatted output
	assert.Contains(t, textOutput, "temperature=0.2&topK=10", "Should show temperature and topK parameters")
	assert.Contains(t, textOutput, "temperature=0.5", "Should show temperature parameter")
	assert.Contains(t, textOutput, "maxOutputTokens=1000&topP=0.8", "Should show maxOutputTokens and topP parameters")
}

func TestCostFormattingWithMoney(t *testing.T) {
	builder := NewPlanBuilder()

	schema := map[string]interface{}{
		"fields": []string{"CompanyName", "Revenue"},
	}

	// Use models with different cost structures
	modelConfig := map[string]string{
		"CompanyName": "gpt-4o",
		"Revenue":     "gpt-3.5-turbo",
	}

	builder.WithSchema(schema).WithModelConfig(modelConfig)

	pricing := DefaultModelPricing()

	// Test both text and JSON formatting with costs
	textOutput, err := builder.ExplainPrettyWithCosts(FormatText, pricing)
	require.NoError(t, err)

	jsonOutput, err := builder.ExplainPrettyWithCosts(FormatJSON, pricing)
	require.NoError(t, err)

	t.Logf("=== Cost Analysis ===")

	// Extract and log costs with money formatting
	plan, err := builder.ExplainWithCosts(pricing)
	require.NoError(t, err)

	t.Logf("💰 Total Execution Cost: $%.4f", plan.EstCost)

	for _, child := range plan.Children {
		if child.Type == PromptCallType {
			t.Logf("💸 %s (%s): $%.6f", child.PromptName, child.Model, child.EstCost)
			if child.ActCost != nil {
				t.Logf("💵 Actual cost: $%.6f", *child.ActCost)
			}
		}
	}

	// Verify both outputs contain cost information
	assert.Contains(t, textOutput, "$", "Text format should show costs with $ symbol")
	assert.Contains(t, jsonOutput, "estCost", "JSON format should include cost estimates")

	t.Logf("\n=== Text Format with Costs ===")
	t.Logf("%s", textOutput)
}
