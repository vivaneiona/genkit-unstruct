package unstruct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test structure that reproduces the edge case from vision example
type EdgeCaseData struct {
	// Same prompt with different models
	DocumentType string `json:"documentType" unstruct:"document-type,gemini-1.5-flash"`
	DocumentDate string `json:"documentDate" unstruct:"document-type,gemini-2.5-pro"`
}

func TestEdgeCaseSamePromptDifferentModels(t *testing.T) {
	sch, err := schemaOf[EdgeCaseData]()
	require.NoError(t, err)

	// Should create 2 separate groups because models are different
	expectedGroups := 2
	assert.Equal(t, expectedGroups, len(sch.group2keys), "Expected %d prompt groups, got %d: %+v", expectedGroups, len(sch.group2keys), sch.group2keys)

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
	assert.Len(t, flashGroup, 1, "Expected 1 flash group key, got %d: %v", len(flashGroup), flashGroup)
	assert.Len(t, proGroup, 1, "Expected 1 pro group key, got %d: %v", len(proGroup), proGroup)

	// Verify the correct fields are in each group
	if len(flashGroup) > 0 {
		assert.Equal(t, "documentType", flashGroup[0], "Expected documentType in flash group, got %v", flashGroup)
	}
	if len(proGroup) > 0 {
		assert.Equal(t, "documentDate", proGroup[0], "Expected documentDate in pro group, got %v", proGroup)
	}

	// Verify field specs have correct models
	typeSpec, exists := sch.json2field["documentType"]
	require.True(t, exists, "Expected documentType field spec")
	assert.Equal(t, "gemini-1.5-flash", typeSpec.model, "Expected documentType model to be gemini-1.5-flash, got %q", typeSpec.model)

	dateSpec, exists := sch.json2field["documentDate"]
	require.True(t, exists, "Expected documentDate field spec")
	assert.Equal(t, "gemini-2.5-pro", dateSpec.model, "Expected documentDate model to be gemini-2.5-pro, got %q", dateSpec.model)
}
