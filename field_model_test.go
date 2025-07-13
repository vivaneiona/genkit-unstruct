package unstruct

import (
	"testing"
)

// TestStruct for testing field-specific model overrides
type TestStruct struct {
	Name        string `json:"name" unstruct:"basic"`
	Description string `json:"description" unstruct:"basic"`
	Details     string `json:"details" unstruct:"advanced"`
}

func TestWithModelFor_FieldSpecificModels(t *testing.T) {
	// Create options with field-specific model overrides
	var opts Options

	// Apply the field model override
	WithModelFor("gemini-1.5-flash", TestStruct{}, "Name")(&opts)
	WithModelFor("gemini-1.5-pro", TestStruct{}, "Description")(&opts)

	// Verify the field models map was created and populated
	if opts.FieldModels == nil {
		t.Fatal("FieldModels map was not created")
	}

	expectedMappings := map[string]string{
		"TestStruct.Name":        "gemini-1.5-flash",
		"TestStruct.Description": "gemini-1.5-pro",
	}

	for expectedKey, expectedModel := range expectedMappings {
		if actualModel, exists := opts.FieldModels[expectedKey]; !exists {
			t.Errorf("Expected field mapping for %s not found", expectedKey)
		} else if actualModel != expectedModel {
			t.Errorf("Expected model %s for field %s, got %s", expectedModel, expectedKey, actualModel)
		}
	}
}

func TestSchemaOfWithOptions_FieldModelOverrides(t *testing.T) {
	// Create options with field-specific model overrides
	opts := &Options{
		Model: "gemini-1.5-pro", // default model
		FieldModels: FieldModelMap{
			"TestStruct.Name":        "gemini-1.5-flash",
			"TestStruct.Description": "gemini-2.0-flash-exp",
		},
	}

	// Generate schema with options
	sch, err := schemaOfWithOptions[TestStruct](opts)
	if err != nil {
		t.Fatalf("schemaOfWithOptions failed: %v", err)
	}

	// Verify that field-specific models are applied correctly
	for _, fieldSpec := range sch.json2field {
		switch fieldSpec.jsonKey {
		case "name":
			if fieldSpec.model != "gemini-1.5-flash" {
				t.Errorf("Expected model gemini-1.5-flash for name field, got %s", fieldSpec.model)
			}
		case "description":
			if fieldSpec.model != "gemini-2.0-flash-exp" {
				t.Errorf("Expected model gemini-2.0-flash-exp for description field, got %s", fieldSpec.model)
			}
		case "details":
			// This field should use the default model since no override was specified
			if fieldSpec.model != "" {
				t.Errorf("Expected empty model for details field (should inherit), got %s", fieldSpec.model)
			}
		}
	}
}

func TestSchemaOfWithOptions_PromptGrouping(t *testing.T) {
	// Test that fields with different models are grouped separately
	opts := &Options{
		Model: "gemini-1.5-pro",
		FieldModels: FieldModelMap{
			"TestStruct.Name": "gemini-1.5-flash", // Different model for name
		},
	}

	sch, err := schemaOfWithOptions[TestStruct](opts)
	if err != nil {
		t.Fatalf("schemaOfWithOptions failed: %v", err)
	}

	// We should have at least 2 groups due to different models
	if len(sch.group2keys) < 2 {
		t.Errorf("Expected at least 2 groups due to different models, got %d", len(sch.group2keys))
	}

	// Verify that fields with different models are in separate groups
	var nameGroup, otherGroup *promptKey
	for pk := range sch.group2keys {
		if pk.model == "gemini-1.5-flash" {
			nameGroup = &pk
		} else if pk.prompt == "basic" {
			otherGroup = &pk
		}
	}

	if nameGroup == nil {
		t.Error("Expected a group with gemini-1.5-flash model for name field")
	}
	if otherGroup == nil {
		t.Error("Expected a group with basic prompt for other fields")
	}
}

func TestWithModelFor_ChainedCalls(t *testing.T) {
	// Test that multiple WithModelFor calls can be chained
	var opts Options

	WithModelFor("model1", TestStruct{}, "Name")(&opts)
	WithModelFor("model2", TestStruct{}, "Description")(&opts)
	WithModelFor("model3", TestStruct{}, "Details")(&opts)

	expected := map[string]string{
		"TestStruct.Name":        "model1",
		"TestStruct.Description": "model2",
		"TestStruct.Details":     "model3",
	}

	for key, expectedModel := range expected {
		if actualModel, exists := opts.FieldModels[key]; !exists {
			t.Errorf("Expected field mapping for %s not found", key)
		} else if actualModel != expectedModel {
			t.Errorf("Expected model %s for field %s, got %s", expectedModel, key, actualModel)
		}
	}

	if len(opts.FieldModels) != 3 {
		t.Errorf("Expected 3 field model mappings, got %d", len(opts.FieldModels))
	}
}
