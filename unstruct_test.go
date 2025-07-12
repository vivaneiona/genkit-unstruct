package unstruct

import (
	"context"
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
	tag2keys, json2field := schemaOf[TestProject]()

	// Check that we have the expected tags
	if len(tag2keys) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tag2keys))
	}

	// Check basic tag
	basicKeys := tag2keys["basic"]
	if len(basicKeys) != 2 || basicKeys[0] != "name" || basicKeys[1] != "code" {
		t.Errorf("Expected basic keys [name, code], got %v", basicKeys)
	}

	// Check coords tag
	coordKeys := tag2keys["coords"]
	if len(coordKeys) != 2 || coordKeys[0] != "lat" || coordKeys[1] != "lon" {
		t.Errorf("Expected coords keys [lat, lon], got %v", coordKeys)
	}

	// Check json2field mapping
	if len(json2field) != 4 {
		t.Errorf("Expected 4 field mappings, got %d", len(json2field))
	}

	if field, ok := json2field["name"]; !ok || field.Name != "Name" {
		t.Errorf("Expected field mapping for 'name' to 'Name', got %v", field)
	}
}

func TestUnstructor_CallPrompt(t *testing.T) {
	ext := NewForTesting[TestProject](mockPrompts{})

	keys := []string{"name", "code"}
	doc := "Project Alpha with code ABC-123"

	raw, err := ext.callPrompt(
		context.Background(),
		"basic",
		keys,
		doc,
		Options{Model: "test-model"},
	)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(raw) == 0 {
		t.Error("Expected non-empty response")
	}
}

func TestUnstructor_UnstructAll(t *testing.T) {
	ext := NewForTesting[TestProject](mockPrompts{})

	doc := "Project Alpha with code ABC-123. Located at coordinates 40.7128, -74.0060."

	result, err := ext.Unstruct(
		context.Background(),
		doc,
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
