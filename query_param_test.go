package unstruct

import (
	"testing"
)

func TestQueryParameterParsing(t *testing.T) {
	// Test tag with query parameters
	tag := "model/vertex/gemini-1.5-flash?temperature=0.5&topK=10"
	result := parseUnstructTag(tag, "inherited")

	// Check basic parsing
	if result.prompt != "inherited" {
		t.Errorf("Expected prompt 'inherited', got '%s'", result.prompt)
	}
	if result.model != "vertex/gemini-1.5-flash" {
		t.Errorf("Expected model 'vertex/gemini-1.5-flash', got '%s'", result.model)
	}

	// Check parameters
	if result.parameters == nil {
		t.Error("Expected parameters map to be initialized")
	}
	if temp, exists := result.parameters["temperature"]; !exists {
		t.Error("Expected 'temperature' parameter to exist")
	} else if temp != "0.5" {
		t.Errorf("Expected temperature '0.5', got '%s'", temp)
	}
	if topK, exists := result.parameters["topK"]; !exists {
		t.Error("Expected 'topK' parameter to exist")
	} else if topK != "10" {
		t.Errorf("Expected topK '10', got '%s'", topK)
	}
}
