package unstruct

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ComplexNestedProject represents a complex nested structure where models should be defined only once
type ComplexNestedProject struct {
	// Basic fields with different prompts
	ProjectID   string `json:"projectId" unstruct:"project"`
	ProjectName string `json:"projectName" unstruct:"project"`

	// Company information with specific model defined once
	Company struct {
		Name         string `json:"name"`
		Address      string `json:"address"`
		Registration string `json:"registration"`
		Phone        string `json:"phone"`
	} `json:"company" unstruct:"company-info,gemini-1.5-pro"`

	// Participant nested structure with model defined at parent level
	Participants []struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Role       string `json:"role"`
		Department string `json:"department"`
		// Nested address within participant
		Address struct {
			Street  string `json:"street"`
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"address"`
	} `json:"participants" unstruct:"participant-info,gemini-1.5-flash"`

	// Financial data with specific model
	FinancialInfo struct {
		Budget       float64 `json:"budget"`
		Currency     string  `json:"currency"`
		TaxRate      float64 `json:"taxRate"`
		PaymentTerms string  `json:"paymentTerms"`
		// Nested cost breakdown
		CostBreakdown struct {
			Labor     float64 `json:"labor"`
			Materials float64 `json:"materials"`
			Overhead  float64 `json:"overhead"`
		} `json:"costBreakdown"`
	} `json:"financialInfo" unstruct:"financial,gemini-1.5-pro"`

	// Certificate information
	Certificates []struct {
		Type       string `json:"type"`
		Issuer     string `json:"issuer"`
		ExpiryDate string `json:"expiryDate"`
		Status     string `json:"status"`
	} `json:"certificates" unstruct:"cert-info,gemini-1.5-flash"`

	// Location data (no specific model - should use default/inherited)
	Location struct {
		Latitude  float64 `json:"lat"`
		Longitude float64 `json:"lon"`
		Timezone  string  `json:"timezone"`
	} `json:"location"`
}

// MockDryRunUnstructor implements DryRunner for testing
type MockDryRunUnstructor struct {
	callCount      int
	lastAssets     []Asset
	lastOptions    *Options
	executionStats *ExecutionStats
}

func (m *MockDryRunUnstructor) DryRun(ctx context.Context, assets []Asset, optFns ...func(*Options)) (*ExecutionStats, error) {
	m.callCount++
	m.lastAssets = assets

	// Apply options
	opts := &Options{}
	for _, fn := range optFns {
		fn(opts)
	}
	m.lastOptions = opts

	// Return predefined execution stats
	if m.executionStats != nil {
		return m.executionStats, nil
	}

	// Default execution stats based on ComplexNestedProject schema
	return &ExecutionStats{
		PromptCalls: 5, // project, company-info, participant-info, financial, cert-info
		ModelCalls: map[string]int{
			"gemini-1.5-pro":   2, // company-info, financial
			"gemini-1.5-flash": 2, // participant-info, cert-info
			"gpt-3.5-turbo":    1, // project (default model)
		},
		PromptGroups:      5,
		FieldsExtracted:   20, // Approximate number of fields
		TotalInputTokens:  1000,
		TotalOutputTokens: 500,
		GroupDetails: []GroupExecution{
			{
				PromptName:   "project",
				Model:        "gpt-3.5-turbo",
				Fields:       []string{"projectId", "projectName"},
				InputTokens:  100,
				OutputTokens: 50,
			},
			{
				PromptName:   "company-info",
				Model:        "gemini-1.5-pro",
				Fields:       []string{"company.name", "company.address", "company.registration", "company.phone"},
				InputTokens:  200,
				OutputTokens: 100,
			},
			{
				PromptName:   "participant-info",
				Model:        "gemini-1.5-flash",
				Fields:       []string{"participants.name", "participants.email", "participants.role", "participants.department", "participants.address.street", "participants.address.city", "participants.address.country"},
				InputTokens:  300,
				OutputTokens: 150,
			},
			{
				PromptName:   "financial",
				Model:        "gemini-1.5-pro",
				Fields:       []string{"financialInfo.budget", "financialInfo.currency", "financialInfo.taxRate", "financialInfo.paymentTerms", "financialInfo.costBreakdown.labor", "financialInfo.costBreakdown.materials", "financialInfo.costBreakdown.overhead"},
				InputTokens:  250,
				OutputTokens: 125,
			},
			{
				PromptName:   "cert-info",
				Model:        "gemini-1.5-flash",
				Fields:       []string{"certificates.type", "certificates.issuer", "certificates.expiryDate", "certificates.status"},
				InputTokens:  150,
				OutputTokens: 75,
			},
		},
	}, nil
}

func TestComplexNestedStructure_ModelDefinedOnceInParent(t *testing.T) {
	sch, err := schemaOf[ComplexNestedProject]()
	require.NoError(t, err)

	// Verify that we have the expected number of prompt groups
	// project, company-info, participant-info (2 groups for nested address), financial (2 groups for nested cost breakdown), cert-info, location (default)
	expectedGroups := 8
	assert.Equal(t, expectedGroups, len(sch.group2keys), "Expected %d prompt groups", expectedGroups)

	// Test that models are correctly inherited from parent structures
	testCases := []struct {
		field         string
		expectedModel string
		description   string
	}{
		// Company fields should inherit gemini-1.5-pro from parent
		{"company.name", "gemini-1.5-pro", "Company name should inherit model from parent"},
		{"company.address", "gemini-1.5-pro", "Company address should inherit model from parent"},
		{"company.registration", "gemini-1.5-pro", "Company registration should inherit model from parent"},
		{"company.phone", "gemini-1.5-pro", "Company phone should inherit model from parent"},

		// Participant fields should inherit gemini-1.5-flash from parent
		{"participants.name", "gemini-1.5-flash", "Participant name should inherit model from parent"},
		{"participants.email", "gemini-1.5-flash", "Participant email should inherit model from parent"},
		{"participants.role", "gemini-1.5-flash", "Participant role should inherit model from parent"},
		{"participants.department", "gemini-1.5-flash", "Participant department should inherit model from parent"},

		// Nested participant address fields should also inherit
		{"participants.address.street", "gemini-1.5-flash", "Nested participant address street should inherit model"},
		{"participants.address.city", "gemini-1.5-flash", "Nested participant address city should inherit model"},
		{"participants.address.country", "gemini-1.5-flash", "Nested participant address country should inherit model"},

		// Financial fields should inherit gemini-1.5-pro from parent
		{"financialInfo.budget", "gemini-1.5-pro", "Financial budget should inherit model from parent"},
		{"financialInfo.currency", "gemini-1.5-pro", "Financial currency should inherit model from parent"},
		{"financialInfo.taxRate", "gemini-1.5-pro", "Financial tax rate should inherit model from parent"},
		{"financialInfo.paymentTerms", "gemini-1.5-pro", "Financial payment terms should inherit model from parent"},

		// Nested financial cost breakdown fields should also inherit
		{"financialInfo.costBreakdown.labor", "gemini-1.5-pro", "Nested financial labor should inherit model"},
		{"financialInfo.costBreakdown.materials", "gemini-1.5-pro", "Nested financial materials should inherit model"},
		{"financialInfo.costBreakdown.overhead", "gemini-1.5-pro", "Nested financial overhead should inherit model"},

		// Certificate fields should inherit gemini-1.5-flash from parent
		{"certificates.type", "gemini-1.5-flash", "Certificate type should inherit model from parent"},
		{"certificates.issuer", "gemini-1.5-flash", "Certificate issuer should inherit model from parent"},
		{"certificates.expiryDate", "gemini-1.5-flash", "Certificate expiry date should inherit model from parent"},
		{"certificates.status", "gemini-1.5-flash", "Certificate status should inherit model from parent"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fieldSpec, exists := sch.json2field[tc.field]
			require.True(t, exists, "Field %s should exist in schema", tc.field)
			assert.Equal(t, tc.expectedModel, fieldSpec.model, tc.description)
		})
	}
}

func TestComplexNestedStructure_GroupingByPromptAndModel(t *testing.T) {
	sch, err := schemaOf[ComplexNestedProject]()
	require.NoError(t, err)

	// Verify that fields are correctly grouped by prompt and model
	testGroups := []struct {
		expectedPrompt string
		expectedModel  string
		expectedFields []string
		description    string
	}{
		{
			expectedPrompt: "project",
			expectedModel:  "",
			expectedFields: []string{"projectId", "projectName"},
			description:    "Project fields should be grouped together",
		},
		{
			expectedPrompt: "company-info",
			expectedModel:  "gemini-1.5-pro",
			expectedFields: []string{"company.name", "company.address", "company.registration", "company.phone"},
			description:    "Company fields should be grouped with gemini-1.5-pro model",
		},
		{
			expectedPrompt: "financial",
			expectedModel:  "gemini-1.5-pro",
			expectedFields: []string{"financialInfo.budget", "financialInfo.currency", "financialInfo.taxRate", "financialInfo.paymentTerms"},
			description:    "Financial root fields should be grouped with gemini-1.5-pro model",
		},
		{
			expectedPrompt: "cert-info",
			expectedModel:  "gemini-1.5-flash",
			expectedFields: []string{"certificates.type", "certificates.issuer", "certificates.expiryDate", "certificates.status"},
			description:    "Certificate fields should be grouped with gemini-1.5-flash model",
		},
	}

	for _, tg := range testGroups {
		t.Run(tg.description, func(t *testing.T) {
			found := false
			for pk, fields := range sch.group2keys {
				if pk.prompt == tg.expectedPrompt && pk.model == tg.expectedModel {
					// For financial group, we need to check specific parent path to distinguish
					// between root fields and nested cost breakdown fields
					if tg.expectedPrompt == "financial" {
						// Check if this is the root financial group (parent path is "financialInfo")
						if pk.parentPath == "financialInfo" {
							found = true
							assert.ElementsMatch(t, tg.expectedFields, fields, tg.description)
							break
						}
					} else {
						found = true
						assert.ElementsMatch(t, tg.expectedFields, fields, tg.description)
						break
					}
				}
			}
			assert.True(t, found, "Expected prompt group not found: %s with model %s", tg.expectedPrompt, tg.expectedModel)
		})
	}
}

func TestDryRun_SingleCallVerification(t *testing.T) {
	mockUnstructor := &MockDryRunUnstructor{}

	// Create a plan builder with our mock unstructor
	builder := NewPlanBuilder()
	builder.WithUnstructor(mockUnstructor)
	builder.WithSampleDocument("test document content")

	// Execute dry run
	stats, err := builder.callDryRun()
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify that DryRun was called exactly once
	assert.Equal(t, 1, mockUnstructor.callCount, "DryRun should be called exactly once")

	// Verify that the assets were passed correctly
	assert.Len(t, mockUnstructor.lastAssets, 1, "Should have exactly one asset")
	textAsset, ok := mockUnstructor.lastAssets[0].(*TextAsset)
	assert.True(t, ok, "Asset should be a TextAsset")
	assert.Equal(t, "test document content", textAsset.Content, "Asset content should match")

	// Verify that the default model option was applied
	assert.NotNil(t, mockUnstructor.lastOptions, "Options should be passed to DryRun")
	assert.Equal(t, "gpt-3.5-turbo", mockUnstructor.lastOptions.Model, "Default model should be gpt-3.5-turbo")
}

func TestDryRun_ComplexNestedStructure_ExecutionStats(t *testing.T) {
	mockUnstructor := &MockDryRunUnstructor{
		executionStats: &ExecutionStats{
			PromptCalls: 5,
			ModelCalls: map[string]int{
				"gemini-1.5-pro":   2, // company-info, financial
				"gemini-1.5-flash": 2, // participant-info, cert-info
				"gpt-3.5-turbo":    1, // project
			},
			PromptGroups:      5,
			FieldsExtracted:   20,
			TotalInputTokens:  1000,
			TotalOutputTokens: 500,
			GroupDetails: []GroupExecution{
				{
					PromptName:   "company-info",
					Model:        "gemini-1.5-pro",
					Fields:       []string{"company.name", "company.address", "company.registration", "company.phone"},
					InputTokens:  200,
					OutputTokens: 100,
				},
				{
					PromptName: "financial",
					Model:      "gemini-1.5-pro",
					Fields: []string{
						"financialInfo.budget", "financialInfo.currency", "financialInfo.taxRate", "financialInfo.paymentTerms",
						"financialInfo.costBreakdown.labor", "financialInfo.costBreakdown.materials", "financialInfo.costBreakdown.overhead",
					},
					InputTokens:  250,
					OutputTokens: 125,
				},
			},
		},
	}

	builder := NewPlanBuilder()
	builder.WithUnstructor(mockUnstructor)
	builder.WithSampleDocument("complex nested document")

	stats, err := builder.callDryRun()
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify execution stats
	assert.Equal(t, 5, stats.PromptCalls, "Should have 5 prompt calls")
	assert.Equal(t, 5, stats.PromptGroups, "Should have 5 prompt groups")
	assert.Equal(t, 20, stats.FieldsExtracted, "Should extract 20 fields")
	assert.Equal(t, 1000, stats.TotalInputTokens, "Should have 1000 input tokens")
	assert.Equal(t, 500, stats.TotalOutputTokens, "Should have 500 output tokens")

	// Verify model call distribution
	require.NotNil(t, stats.ModelCalls, "ModelCalls should not be nil")
	assert.Equal(t, 2, stats.ModelCalls["gemini-1.5-pro"], "Should have 2 calls to gemini-1.5-pro")
	assert.Equal(t, 2, stats.ModelCalls["gemini-1.5-flash"], "Should have 2 calls to gemini-1.5-flash")
	assert.Equal(t, 1, stats.ModelCalls["gpt-3.5-turbo"], "Should have 1 call to gpt-3.5-turbo")

	// Verify group details
	require.Len(t, stats.GroupDetails, 2, "Should have 2 group details")

	// Check company-info group
	companyGroup := stats.GroupDetails[0]
	assert.Equal(t, "company-info", companyGroup.PromptName)
	assert.Equal(t, "gemini-1.5-pro", companyGroup.Model)
	assert.Len(t, companyGroup.Fields, 4, "Company group should have 4 fields")
	assert.Contains(t, companyGroup.Fields, "company.name")
	assert.Contains(t, companyGroup.Fields, "company.address")
	assert.Contains(t, companyGroup.Fields, "company.registration")
	assert.Contains(t, companyGroup.Fields, "company.phone")

	// Check financial group
	financialGroup := stats.GroupDetails[1]
	assert.Equal(t, "financial", financialGroup.PromptName)
	assert.Equal(t, "gemini-1.5-pro", financialGroup.Model)
	assert.Len(t, financialGroup.Fields, 7, "Financial group should have 7 fields including nested ones")
	assert.Contains(t, financialGroup.Fields, "financialInfo.budget")
	assert.Contains(t, financialGroup.Fields, "financialInfo.costBreakdown.labor")
	assert.Contains(t, financialGroup.Fields, "financialInfo.costBreakdown.materials")
	assert.Contains(t, financialGroup.Fields, "financialInfo.costBreakdown.overhead")
}

func TestDryRun_NoRedundantCalls(t *testing.T) {
	mockUnstructor := &MockDryRunUnstructor{}

	builder := NewPlanBuilder()
	builder.WithUnstructor(mockUnstructor)
	builder.WithSampleDocument("test document")

	// Call dry run multiple times
	stats1, err1 := builder.callDryRun()
	require.NoError(t, err1)
	require.NotNil(t, stats1)

	stats2, err2 := builder.callDryRun()
	require.NoError(t, err2)
	require.NotNil(t, stats2)

	stats3, err3 := builder.callDryRun()
	require.NoError(t, err3)
	require.NotNil(t, stats3)

	// Each call to callDryRun should result in exactly one call to the underlying DryRun method
	assert.Equal(t, 3, mockUnstructor.callCount, "DryRun should be called exactly once per callDryRun invocation")
}

func TestComplexNestedStructure_FieldModelOverrides(t *testing.T) {
	// Test that field-specific model overrides work correctly with nested structures
	opts := &Options{
		FieldModels: FieldModelMap{
			"ComplexNestedProject.Company":       "custom-model-1",
			"ComplexNestedProject.FinancialInfo": "custom-model-2",
		},
	}

	sch, err := schemaOfWithOptions[ComplexNestedProject](opts)
	require.NoError(t, err)

	// Find fields that should have custom models
	companyNameSpec, exists := sch.json2field["company.name"]
	require.True(t, exists, "company.name field should exist")
	// Note: The current implementation sets field-level overrides based on exact field matches
	// The nested Company struct gets the override applied to the entire nested structure
	assert.Equal(t, "custom-model-1", companyNameSpec.model, "Company name should use custom model override")
}

func TestDryRun_ErrorHandling(t *testing.T) {
	// Test DryRun error handling
	builder := NewPlanBuilder()

	// Test with no unstructor
	_, err := builder.callDryRun()
	assert.Error(t, err, "Should error when no unstructor is set")
	assert.Contains(t, err.Error(), "unstructor or sample document not configured")

	// Test with unstructor that doesn't implement DryRunner
	builder.WithUnstructor("not a dry runner")
	builder.WithSampleDocument("test document")
	_, err = builder.callDryRun()
	assert.Error(t, err, "Should error when unstructor doesn't implement DryRunner")
	assert.Contains(t, err.Error(), "does not implement DryRunner interface")

	// Test with no document
	mockUnstructor := &MockDryRunUnstructor{}
	builder.WithUnstructor(mockUnstructor)
	builder.WithSampleDocument("") // Reset document
	_, err = builder.callDryRun()
	assert.Error(t, err, "Should error when no document is provided")
	assert.Contains(t, err.Error(), "unstructor or sample document not configured")
}
