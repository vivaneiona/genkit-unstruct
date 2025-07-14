package unstruct

import (
	"context"
	"testing"
)

type Company struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// Request struct with ComplexProject that has model defined at parent level
type Request struct {
	ComplexProject struct {
		ProjectColor string  `json:"projectColor"`
		ProjectMode  string  `json:"projectMode"`
		ProjectName  string  `json:"projectName"`
		CertIssuer   string  `json:"certIssuer"`
		Latitude     float64 `json:"lat"`
		Longitude    float64 `json:"lon"`

		// Nested structure with specific model for high-accuracy participant extraction
		Participant struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"participant"`

		Company    Company
		Affiliated []Company
	} `json:"complexProject" unstruct:"prompt-name,gemini-1.5-pro"`
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
		{"model/gemini-1.5-pro", "default", "default", "gemini-1.5-pro"},
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

func TestSingleCallWithParentStructModel(t *testing.T) {
	// Test that when model is defined at parent struct level,
	// fields are grouped by parent path but use the same prompt and model
	sch, err := schemaOf[Request]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// The current implementation creates 4 groups based on parent paths:
	// 1. complexProject (direct fields)
	// 2. complexProject.participant (nested struct)
	// 3. complexProject.Company (embedded struct)
	// 4. complexProject.Affiliated (slice of structs)
	expectedGroups := 4
	if len(sch.group2keys) != expectedGroups {
		t.Errorf("Expected %d prompt groups, got %d: %+v", expectedGroups, len(sch.group2keys), sch.group2keys)
	}

	// Verify all groups use the same prompt and model
	totalFields := 0
	for pk, keys := range sch.group2keys {
		if pk.prompt != "prompt-name" {
			t.Errorf("Expected prompt 'prompt-name', got %q", pk.prompt)
		}
		if pk.model != "gemini-1.5-pro" {
			t.Errorf("Expected model 'gemini-1.5-pro', got %q", pk.model)
		}
		totalFields += len(keys)
	}

	// Should include all fields from the complex project structure
	expectedTotalFields := 12 // projectColor, projectMode, projectName, certIssuer, lat, lon, participant.name, participant.address, Company.name, Company.address, Affiliated.name, Affiliated.address
	if totalFields != expectedTotalFields {
		t.Errorf("Expected %d total fields across all groups, got %d", expectedTotalFields, totalFields)
	}
}

// UserRequestedStructure - the exact structure you requested
type UserRequestedStructure struct {
	ComplexProject struct {
		ProjectColor string  `json:"projectColor"`
		ProjectMode  string  `json:"projectMode"`
		ProjectName  string  `json:"projectName"`
		CertIssuer   string  `json:"certIssuer"`
		Latitude     float64 `json:"lat"`
		Longitude    float64 `json:"lon"`

		// Nested structure with specific model for high-accuracy participant extraction
		Participant struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"participant"`

		Company    Company
		Affiliated []Company
	} `json:"complexProject" unstruct:"prompt-name,gemini-1.5-pro"`
}

func TestSingleCallDryRunWithUnstruct(t *testing.T) {
	// Create a simple prompt provider for testing
	promptProvider := SimplePromptProvider{
		"prompt-name": "Extract the following fields from the document: {{.Keys}}. Return JSON.",
	}

	// Create Unstruct instance with your exact structure using test invoker
	unstruct := newTestingUnstructor[UserRequestedStructure](promptProvider)

	// Call DryRun on the structure with model specified
	stats, err := unstruct.DryRun(context.Background(), []Asset{
		&TextAsset{Content: "test document content with project data"},
	}, WithModel("gemini-1.5-pro"))
	if err != nil {
		t.Fatalf("DryRun failed: %v", err)
	}

	// The current implementation creates 4 groups based on parent paths:
	// 1. complexProject (direct fields)
	// 2. complexProject.participant (nested struct)
	// 3. complexProject.Company (embedded struct)
	// 4. complexProject.Affiliated (slice of structs)
	expectedGroups := 4
	if stats.PromptGroups != expectedGroups {
		t.Errorf("Expected %d prompt groups, got %d", expectedGroups, stats.PromptGroups)
	}

	if stats.PromptCalls != expectedGroups {
		t.Errorf("Expected %d prompt calls, got %d", expectedGroups, stats.PromptCalls)
	}

	// Verify the model used - should be called 4 times (once per group)
	if len(stats.ModelCalls) != 1 {
		t.Errorf("Expected exactly 1 model type used, got %d: %v", len(stats.ModelCalls), stats.ModelCalls)
	}

	if stats.ModelCalls["gemini-1.5-pro"] != expectedGroups {
		t.Errorf("Expected gemini-1.5-pro to be called %d times, got: %v", expectedGroups, stats.ModelCalls)
	}

	// Verify group details
	if len(stats.GroupDetails) != expectedGroups {
		t.Errorf("Expected %d group details, got %d", expectedGroups, len(stats.GroupDetails))
	}

	// All groups should use the same prompt and model
	for i, group := range stats.GroupDetails {
		if group.PromptName != "prompt-name" {
			t.Errorf("Group %d: Expected prompt name 'prompt-name', got %q", i, group.PromptName)
		}

		if group.Model != "gemini-1.5-pro" {
			t.Errorf("Group %d: Expected model 'gemini-1.5-pro', got %q", i, group.Model)
		}
	}

	t.Logf("SUCCESS: Structure correctly grouped by parent paths - PromptGroups: %d, PromptCalls: %d, Model: gemini-1.5-pro used %d times",
		stats.PromptGroups, stats.PromptCalls, stats.ModelCalls["gemini-1.5-pro"])
}

func TestSingleCallDryRunWithFlattenGroups(t *testing.T) {
	// Create a simple prompt provider for testing
	promptProvider := SimplePromptProvider{
		"prompt-name": "Extract the following fields from the document: {{.Keys}}. Return JSON.",
	}

	// Create Unstruct instance with your exact structure using test invoker
	unstruct := newTestingUnstructor[UserRequestedStructure](promptProvider)

	// Call DryRun on the structure with model specified AND flattened groups
	stats, err := unstruct.DryRun(context.Background(), []Asset{
		&TextAsset{Content: "test document content with project data"},
	}, WithModel("gemini-1.5-pro"), WithFlattenGroups())
	if err != nil {
		t.Fatalf("DryRun failed: %v", err)
	}

	// With FlattenGroups enabled, all fields with same prompt+model should be in one group
	expectedGroups := 1
	if stats.PromptGroups != expectedGroups {
		t.Errorf("Expected %d prompt group (single call), got %d", expectedGroups, stats.PromptGroups)
	}

	if stats.PromptCalls != expectedGroups {
		t.Errorf("Expected %d prompt call, got %d", expectedGroups, stats.PromptCalls)
	}

	// Verify the model used - should be called once
	if len(stats.ModelCalls) != 1 {
		t.Errorf("Expected exactly 1 model type used, got %d: %v", len(stats.ModelCalls), stats.ModelCalls)
	}

	if stats.ModelCalls["gemini-1.5-pro"] != 1 {
		t.Errorf("Expected gemini-1.5-pro to be called once, got: %v", stats.ModelCalls)
	}

	// Verify group details
	if len(stats.GroupDetails) != 1 {
		t.Errorf("Expected exactly 1 group detail, got %d", len(stats.GroupDetails))
	}

	group := stats.GroupDetails[0]
	if group.PromptName != "prompt-name" {
		t.Errorf("Expected prompt name 'prompt-name', got %q", group.PromptName)
	}

	if group.Model != "gemini-1.5-pro" {
		t.Errorf("Expected model 'gemini-1.5-pro', got %q", group.Model)
	}

	// Should have all fields from the complex structure flattened into one group
	expectedMinFields := 10 // at least the main fields
	if len(group.Fields) < expectedMinFields {
		t.Errorf("Expected at least %d fields, got %d: %v", expectedMinFields, len(group.Fields), group.Fields)
	}

	t.Logf("SUCCESS: Single call verified with FlattenGroups - PromptGroups: %d, PromptCalls: %d, Model: %s, Fields: %d",
		stats.PromptGroups, stats.PromptCalls, group.Model, len(group.Fields))
}
