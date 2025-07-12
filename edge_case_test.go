package unstruct

import (
	"testing"
)

// Test structure that reproduces the edge case from vision example
type EdgeCaseData struct {
	// Same prompt with different models
	DocumentType string `json:"documentType" unstruct:"document-type,gemini-1.5-flash"`
	DocumentDate string `json:"documentDate" unstruct:"document-type,gemini-2.5-pro"`
}

func TestEdgeCaseSamePromptDifferentModels(t *testing.T) {
	sch, err := schemaOf[EdgeCaseData]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Should create 2 separate groups because models are different
	expectedGroups := 2
	if len(sch.group2keys) != expectedGroups {
		t.Errorf("Expected %d prompt groups, got %d: %+v", expectedGroups, len(sch.group2keys), sch.group2keys)
	}

	// Verify the grouping - should have separate keys for each model
	var flashGroup []string
	var proGroup []string

	for pk, keys := range sch.group2keys {
		if pk.prompt == "document-type" && pk.model == "gemini-1.5-flash" {
			flashGroup = keys
		} else if pk.prompt == "document-type" && pk.model == "gemini-2.5-pro" {
			proGroup = keys
		}
	}

	// Each group should have exactly 1 field
	if len(flashGroup) != 1 {
		t.Errorf("Expected 1 flash group key, got %d: %v", len(flashGroup), flashGroup)
	}
	if len(proGroup) != 1 {
		t.Errorf("Expected 1 pro group key, got %d: %v", len(proGroup), proGroup)
	}

	// Verify the correct fields are in each group
	if len(flashGroup) > 0 && flashGroup[0] != "documentType" {
		t.Errorf("Expected documentType in flash group, got %v", flashGroup)
	}
	if len(proGroup) > 0 && proGroup[0] != "documentDate" {
		t.Errorf("Expected documentDate in pro group, got %v", proGroup)
	}

	// Verify field specs have correct models
	typeSpec, exists := sch.json2field["documentType"]
	if !exists {
		t.Error("Expected documentType field spec")
	} else if typeSpec.model != "gemini-1.5-flash" {
		t.Errorf("Expected documentType model to be gemini-1.5-flash, got %q", typeSpec.model)
	}

	dateSpec, exists := sch.json2field["documentDate"]
	if !exists {
		t.Error("Expected documentDate field spec")
	} else if dateSpec.model != "gemini-2.5-pro" {
		t.Errorf("Expected documentDate model to be gemini-2.5-pro, got %q", dateSpec.model)
	}
}
