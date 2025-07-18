package unstruct

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Test structs
type TestProject struct {
	Name      string  `json:"name" unstruct:"basic"`
	Code      string  `json:"code" unstruct:"basic"`
	Latitude  float64 `json:"lat" unstruct:"coords"`
	Longitude float64 `json:"lon" unstruct:"coords"`
}

// Test structure with missing prompts
type ProjectWithMissingPrompts struct {
	Name      string  `json:"name" unstruct:"basic"`
	Code      string  `json:"code"` // Missing prompt - should error
	Latitude  float64 `json:"lat"`  // Missing prompt - should error
	Longitude float64 `json:"lon" unstruct:"coords"`
}

// Mock prompt provider
type mockPrompts struct{}

func (m mockPrompts) GetPrompt(tag string, version int) (string, error) {
	prompts := map[string]string{
		"basic":  "Extract {{.Keys}} from the text as JSON.",
		"coords": "Find {{.Keys}} coordinates in the text as JSON.",
	}
	if prompt, ok := prompts[tag]; ok {
		return prompt, nil
	}
	return "", nil
}

func TestUnstructor_SchemaReflection(t *testing.T) {
	sch, err := schemaOf[TestProject]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Check that we have the expected prompt groups
	if len(sch.group2keys) != 2 {
		t.Errorf("Expected 2 prompt groups, got %d", len(sch.group2keys))
	}

	// Check field mapping
	if len(sch.json2field) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(sch.json2field))
	}

	// Find basic group
	var basicKeys []string
	var coordKeys []string
	for pk, keys := range sch.group2keys {
		if pk.prompt == "basic" {
			basicKeys = keys
		} else if pk.prompt == "coords" {
			coordKeys = keys
		}
	}

	// Check basic keys
	if len(basicKeys) != 2 {
		t.Errorf("Expected 2 basic keys, got %d: %v", len(basicKeys), basicKeys)
	}

	// Check coords keys
	if len(coordKeys) != 2 {
		t.Errorf("Expected 2 coord keys, got %d: %v", len(coordKeys), coordKeys)
	}
}

func TestUnstructor_CallPrompt(t *testing.T) {
	ext := newTestingUnstructor[TestProject](mockPrompts{})

	keys := []string{"name", "code"}
	doc := "Project Alpha with code ABC-123"
	assets := []Asset{&TextAsset{Content: doc}}

	raw, err := ext.callPrompt(
		context.Background(),
		"basic",
		keys,
		assets,
		"test-model",
		nil, // no parameters
		Options{},
	)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(raw) == 0 {
		t.Error("Expected non-empty response")
	}
}

func TestUnstructor_UnstructAll(t *testing.T) {
	ext := newTestingUnstructor[TestProject](mockPrompts{})

	doc := "Project Alpha with code ABC-123. Located at coordinates 40.7128, -74.0060."
	assets := []Asset{&TextAsset{Content: doc}}

	result, err := ext.Unstruct(
		context.Background(),
		assets,
		WithModel("test-model"),
		WithTimeout(10*time.Second),
	)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	// With our test invoker, we should get the mock data
	if result.Name != "Test Project" {
		t.Errorf("Expected name 'Test Project', got %q", result.Name)
	}

	if result.Code != "TEST-123" {
		t.Errorf("Expected code 'TEST-123', got %q", result.Code)
	}
}

func TestSanitizeJSONResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			input:    "  {\"key\": \"value\"}  ",
			expected: "{\"key\": \"value\"}",
		},
		{
			input:    "{\"key\": \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
	}

	for _, test := range tests {
		result := string(SanitizeJSONResponse([]byte(test.input)))
		if result != test.expected {
			t.Errorf("For input %q, expected %q, got %q", test.input, test.expected, result)
		}
	}
}

func TestUnstructor_RequiredPrompts(t *testing.T) {
	ext := newTestingUnstructor[ProjectWithMissingPrompts](mockPrompts{})

	doc := "Project Alpha with code ABC-123. Located at coordinates 40.7128, -74.0060."
	assets := []Asset{&TextAsset{Content: doc}}

	// Should error because fields are missing prompts and no fallback is provided
	result, err := ext.Unstruct(
		context.Background(),
		assets,
		WithModel("test-model"),
		WithTimeout(10*time.Second),
	)

	if err == nil {
		t.Error("Expected error for missing prompts, got nil")
	}

	if result != nil {
		t.Error("Expected nil result when error occurs")
	}

	// Check that the error message mentions missing prompts
	expectedError := "no prompt specified"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, got: %v", expectedError, err)
	}
}

func TestUnstructor_WithFallbackPrompt(t *testing.T) {
	ext := newTestingUnstructor[ProjectWithMissingPrompts](mockPrompts{})

	doc := "Project Alpha with code ABC-123. Located at coordinates 40.7128, -74.0060."
	assets := []Asset{&TextAsset{Content: doc}}

	// Should succeed when fallback prompt is provided
	result, err := ext.Unstruct(
		context.Background(),
		assets,
		WithModel("test-model"),
		WithTimeout(10*time.Second),
		WithFallbackPrompt("basic"), // Explicit fallback
	)

	if err != nil {
		t.Errorf("Expected no error with fallback prompt, got: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result with fallback prompt")
	}
}
