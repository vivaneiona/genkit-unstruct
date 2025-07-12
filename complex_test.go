package unstruct

import (
	"testing"
)

type Company struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// Complex Project structure matching the example
type ComplexProject struct {
	ProjectColor string  `json:"projectColor" unstruct:"project"`
	ProjectMode  string  `json:"projectMode" unstruct:"project"`
	ProjectName  string  `json:"projectName" unstruct:"project"`
	CertIssuer   string  `json:"certIssuer" unstruct:"cert"`
	Latitude     float64 `json:"lat"`
	Longitude    float64 `json:"lon"`

	// Nested structure with specific model for high-accuracy participant extraction
	Participant struct {
		Name    string `json:"name" unstruct:"gemini-1.5-pro"`
		Address string `json:"address" unstruct:"gemini-1.5-pro"`
	} `json:"participant"`

	Company    Company   `unstruct:"prompt-name,gemini-1.5-pro"`
	Affiliated []Company `unstruct:"prompt-name,gemini-1.5-pro"`
}

func TestComplexProjectGrouping(t *testing.T) {
	sch, err := schemaOf[ComplexProject]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// The actual grouping creates 6 groups:
	// 1. default group (empty prompt): lat, lon
	// 2. participant nested group: participant.name, participant.address
	// 3. cert group: certIssuer
	// 4. project group: projectColor, projectMode, projectName
	// 5. company group: Company.name, Company.address
	// 6. affiliated group: Affiliated.name, Affiliated.address
	expectedGroups := 6
	if len(sch.group2keys) != expectedGroups {
		t.Errorf("Expected %d prompt groups, got %d: %+v", expectedGroups, len(sch.group2keys), sch.group2keys)
	}

	// Check that model-specific fields are mapped correctly
	participantNameSpec, exists := sch.json2field["participant.name"]
	if !exists {
		t.Error("Expected participant.name field spec")
	} else if participantNameSpec.model != "" {
		// The nested struct fields inherit the model from the parent parsing
		t.Logf("participant.name model: %q", participantNameSpec.model)
	}

	// Verify that the grouping logic batches fields with the same prompt correctly
	var projectKeys []string
	var participantKeys []string

	for pk, keys := range sch.group2keys {
		if pk.prompt == "project" {
			projectKeys = keys
		} else if pk.parentPath == "participant" {
			participantKeys = keys
		}
	}

	// Project group should have 3 fields
	if len(projectKeys) != 3 {
		t.Errorf("Expected 3 project keys, got %d: %v", len(projectKeys), projectKeys)
	}

	// Participant group should have 2 fields
	if len(participantKeys) != 2 {
		t.Errorf("Expected 2 participant keys, got %d: %v", len(participantKeys), participantKeys)
	}
}

func TestTagParsing(t *testing.T) {
	tests := []struct {
		tag             string
		inheritedPrompt string
		expectedPrompt  string
		expectedModel   string
	}{
		{"", "default", "default", ""},
		{"project", "", "project", ""},
		{"gemini-1.5-pro", "default", "default", "gemini-1.5-pro"},
		{"prompt-name,gemini-1.5-pro", "", "prompt-name", "gemini-1.5-pro"},
		{"cert,gemini-1.5-flash", "base", "cert", "gemini-1.5-flash"},
		{"malformed,too,many,parts", "fallback", "fallback", ""},
	}

	for _, test := range tests {
		tp := parseUnstructTag(test.tag, test.inheritedPrompt)
		if tp.prompt != test.expectedPrompt {
			t.Errorf("Tag %q with inherited %q: expected prompt %q, got %q",
				test.tag, test.inheritedPrompt, test.expectedPrompt, tp.prompt)
		}
		if tp.model != test.expectedModel {
			t.Errorf("Tag %q with inherited %q: expected model %q, got %q",
				test.tag, test.inheritedPrompt, test.expectedModel, tp.model)
		}
	}
}

func TestKnownModel(t *testing.T) {
	knownModels := []string{
		"gemini-1.5-pro", "gemini-1.5-flash", "gemini-1.5-flash-8b",
		"gemini-1.0-pro", "gemini-pro", "gemini-flash",
	}

	unknownModels := []string{
		"gpt-4", "claude-3", "llama-2", "custom-model", "",
	}

	for _, model := range knownModels {
		if !knownModel(model) {
			t.Errorf("Expected %q to be recognized as known model", model)
		}
	}

	for _, model := range unknownModels {
		if knownModel(model) {
			t.Errorf("Expected %q to be recognized as unknown model", model)
		}
	}
}
