package unstruct

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test structures for schema testing
type SimpleStruct struct {
	Name string `json:"name" unstruct:"basic"`
	Age  int    `json:"age" unstruct:"basic"`
}

type NestedStruct struct {
	User    UserInfo `json:"user" unstruct:"user_info"`
	Project struct {
		Name        string `json:"name" unstruct:"project"`
		Description string `json:"desc" unstruct:"project"`
	} `json:"project"`
}

type UserInfo struct {
	FirstName string `json:"first_name" unstruct:"name"`
	LastName  string `json:"last_name" unstruct:"name"`
	Email     string `json:"email" unstruct:"contact"`
}

type SliceStruct struct {
	Tags    []string   `json:"tags" unstruct:"keywords"`
	Users   []UserInfo `json:"users"`
	Numbers []int      `json:"numbers" unstruct:"numeric"`
}

type ModelOverrideStruct struct {
	BasicField string `json:"basic" unstruct:"basic"`
	FlashField string `json:"flash" unstruct:"fast,gemini-1.5-flash"`
	ProField   string `json:"pro" unstruct:"complex,gemini-1.5-pro"`
	ModelOnly  string `json:"model_only" unstruct:"model/gemini-1.5-flash-8b"`
}

type InheritanceStruct struct {
	Parent ParentStruct `json:"parent" unstruct:"inherited_prompt"`
}

type ParentStruct struct {
	Child1 string `json:"child1"`                     // Should inherit "inherited_prompt"
	Child2 string `json:"child2" unstruct:"override"` // Should override
}

type EdgeCaseStruct struct {
	Ignored   string                 `json:"-"`
	NoTag     string                 `json:"no_tag"`
	EmptyTag  string                 `json:"empty_tag" unstruct:""`
	TimeField time.Time              `json:"time_field" unstruct:"temporal"`
	Anonymous struct{ Field string } // Anonymous struct
}

// Test structures for model prefix testing
type ProviderPrefixStruct struct {
	GoogleAIField string `json:"googleai_field" unstruct:"provider_test,googleai/gemini-1.5-pro"`
	PlainField    string `json:"plain_field" unstruct:"plain_test,gemini-1.5-pro"`
	UnknownField  string `json:"unknown_field" unstruct:"unknown_test,custom-model-name"`
	FlashField    string `json:"flash_field" unstruct:"flash_test,gemini-1.5-flash"`
}

// Test structures for tag parsing behavior with new syntax
type TagParsingStruct struct {
	RegularPromptField string `json:"regular_prompt" unstruct:"my_prompt"`
	ModelOnlyField     string `json:"model_only" unstruct:"model/googleai/gemini-1.5-pro"`
	ExplicitField      string `json:"explicit" unstruct:"custom_prompt,googleai/gemini-1.5-pro"`
	EmptyTagField      string `json:"empty_tag" unstruct:""`
}

// Test structures for new syntax formats
type NewSyntaxStruct struct {
	// Traditional explicit format: prompt,model
	ExplicitField string `json:"explicit" unstruct:"my_prompt,my_model"`

	// New model/ prefix format: inherits prompt, specifies model
	ModelOnlyField string `json:"model_only" unstruct:"model/googleai/gemini-1.5-pro"`

	// New prompt/ prefix format: specifies prompt, no model
	PromptOnlyField string `json:"prompt_only" unstruct:"prompt/custom_extraction"`

	// Regular prompt (existing behavior)
	PromptField string `json:"prompt_field" unstruct:"regular_prompt"`

	// Provider-prefixed models with explicit prompt
	ProviderField string `json:"provider_field" unstruct:"extraction,vertex/gemini-1.5-flash"`

	// Complex model names
	ComplexModelField string `json:"complex_model" unstruct:"model/anthropic/claude-3-sonnet"`

	// Empty tag (inherit)
	InheritField string `json:"inherit_field" unstruct:""`
}

func TestSchemaOf_SimpleStruct(t *testing.T) {
	sch, err := schemaOf[SimpleStruct]()
	require.NoError(t, err)

	// Should have one prompt group
	assert.Len(t, sch.group2keys, 1, "Expected 1 prompt group, got %d", len(sch.group2keys))

	// Should have two fields
	assert.Len(t, sch.json2field, 2, "Expected 2 fields, got %d", len(sch.json2field))

	// Check the basic group
	basicKey := promptKey{prompt: "basic", parentPath: "", model: ""}
	keys, exists := sch.group2keys[basicKey]
	assert.True(t, exists, "Expected 'basic' prompt group to exist")
	assert.Len(t, keys, 2, "Expected 2 keys in basic group, got %d", len(keys))

	// Check field specs
	nameField, exists := sch.json2field["name"]
	assert.True(t, exists, "Expected 'name' field to exist")
	assert.Equal(t, "name", nameField.jsonKey, "Expected jsonKey 'name', got %s", nameField.jsonKey)
	assert.Len(t, nameField.index, 1, "Expected index length 1, got %d", len(nameField.index))
}

func TestSchemaOf_NestedStruct(t *testing.T) {
	sch, err := schemaOf[NestedStruct]()
	require.NoError(t, err)

	// Should have multiple prompt groups
	expectedGroups := []promptKey{
		{prompt: "name", parentPath: "user"},
		{prompt: "contact", parentPath: "user"},
		{prompt: "project", parentPath: "project"},
	}

	for _, expectedKey := range expectedGroups {
		_, exists := sch.group2keys[expectedKey]
		assert.True(t, exists, "Expected prompt group %+v to exist", expectedKey)
	}

	// Check nested field paths
	expectedFields := []string{
		"user.first_name",
		"user.last_name",
		"user.email",
		"project.name",
		"project.desc",
	}

	for _, field := range expectedFields {
		_, exists := sch.json2field[field]
		assert.True(t, exists, "Expected field %s to exist", field)
	}
}

func TestSchemaOf_SliceStruct(t *testing.T) {
	sch, err := schemaOf[SliceStruct]()
	require.NoError(t, err)

	// Check that slice of structs is handled correctly
	expectedFields := []string{
		"tags",
		"users.first_name",
		"users.last_name",
		"users.email",
		"numbers",
	}

	for _, field := range expectedFields {
		_, exists := sch.json2field[field]
		assert.True(t, exists, "Expected field %s to exist", field)
	}
}

func TestSchemaOf_ModelOverride(t *testing.T) {
	sch, err := schemaOf[ModelOverrideStruct]()
	require.NoError(t, err)

	// Check model specifications
	flashField := sch.json2field["flash"]
	assert.Equal(t, "gemini-1.5-flash", flashField.model, "Expected model 'gemini-1.5-flash'")

	proField := sch.json2field["pro"]
	assert.Equal(t, "gemini-1.5-pro", proField.model, "Expected model 'gemini-1.5-pro'")

	modelOnlyField := sch.json2field["model_only"]
	assert.Equal(t, "gemini-1.5-flash-8b", modelOnlyField.model, "Expected model 'gemini-1.5-flash-8b'")

	basicField := sch.json2field["basic"]
	assert.Empty(t, basicField.model, "Expected empty model for basic field")
}

func TestSchemaOf_Inheritance(t *testing.T) {
	sch, err := schemaOf[InheritanceStruct]()
	require.NoError(t, err)

	// Check that child1 inherits the prompt
	inheritedKey := promptKey{prompt: "inherited_prompt", parentPath: "parent"}
	keys, exists := sch.group2keys[inheritedKey]
	assert.True(t, exists, "Expected inherited prompt group to exist")

	found := false
	for _, key := range keys {
		if key == "parent.child1" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected parent.child1 to inherit prompt")

	// Check that child2 overrides the prompt
	overrideKey := promptKey{prompt: "override", parentPath: "parent"}
	keys, exists = sch.group2keys[overrideKey]
	assert.True(t, exists, "Expected override prompt group to exist")

	found = false
	for _, key := range keys {
		if key == "parent.child2" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected parent.child2 to override prompt")
}

func TestSchemaOf_EdgeCases(t *testing.T) {
	sch, err := schemaOf[EdgeCaseStruct]()
	require.NoError(t, err)

	// Check ignored fields are not included
	_, exists := sch.json2field["ignored"]
	assert.False(t, exists, "Ignored field should not exist in schema")

	_, exists = sch.json2field["unexported"]
	assert.False(t, exists, "Unexported field should not exist in schema")

	// Check that fields without tags still get included
	_, exists = sch.json2field["no_tag"]
	assert.True(t, exists, "Field without tag should still exist")

	// Check time field is treated as leaf (not struct)
	_, exists = sch.json2field["time_field"]
	assert.True(t, exists, "Time field should exist as leaf")
}

func TestSchemaOf_NonStructType(t *testing.T) {
	// Test with non-struct type
	type StringType string
	_, err := schemaOf[StringType]()
	assert.Error(t, err, "Expected error for non-struct type")
	assert.Equal(t, "unstruct: T must be struct", err.Error(), "Expected specific error message")

	// Test with slice type
	_, err = schemaOf[[]string]()
	assert.Error(t, err, "Expected error for slice type")

	// Test with map type
	_, err = schemaOf[map[string]string]()
	assert.Error(t, err, "Expected error for map type")
}

func TestJoinKey(t *testing.T) {
	tests := []struct {
		parent, child, expected string
	}{
		{"", "child", "child"},
		{"parent", "child", "parent.child"},
		{"grand.parent", "child", "grand.parent.child"},
	}

	for _, test := range tests {
		result := joinKey(test.parent, test.child)
		assert.Equal(t, test.expected, result, "joinKey(%q, %q) = %q, expected %q",
			test.parent, test.child, result, test.expected)
	}
}

func TestIsPureStruct(t *testing.T) {
	tests := []struct {
		typ      reflect.Type
		expected bool
	}{
		{reflect.TypeOf(SimpleStruct{}), true},
		{reflect.TypeOf(UserInfo{}), true},
		{reflect.TypeOf(time.Time{}), false}, // time.Time is special
		{reflect.TypeOf("string"), false},
		{reflect.TypeOf(42), false},
		{reflect.TypeOf([]string{}), false},
		{reflect.TypeOf(map[string]string{}), false},
	}

	for _, test := range tests {
		result := isPureStruct(test.typ)
		assert.Equal(t, test.expected, result, "isPureStruct(%v) = %v, expected %v",
			test.typ, result, test.expected)
	}
}

func TestSchemaGrouping(t *testing.T) {
	type GroupingTest struct {
		Field1 string `json:"f1" unstruct:"group1"`
		Field2 string `json:"f2" unstruct:"group1"`
		Field3 string `json:"f3" unstruct:"group2"`
		Field4 string `json:"f4" unstruct:"group2"`
	}

	sch, err := schemaOf[GroupingTest]()
	require.NoError(t, err)

	// Check group1 has 2 fields
	group1Key := promptKey{prompt: "group1", parentPath: ""}
	keys1, exists := sch.group2keys[group1Key]
	assert.True(t, exists, "Expected group1 to exist")
	assert.Len(t, keys1, 2, "Expected 2 keys in group1")

	// Check group2 has 2 fields
	group2Key := promptKey{prompt: "group2", parentPath: ""}
	keys2, exists := sch.group2keys[group2Key]
	assert.True(t, exists, "Expected group2 to exist")
	assert.Len(t, keys2, 2, "Expected 2 keys in group2")
}

func TestSchemaOf_ProviderPrefixModels(t *testing.T) {
	sch, err := schemaOf[ProviderPrefixStruct]()
	require.NoError(t, err)

	// Check that googleai/ prefixed model is preserved as-is
	googleAIField := sch.json2field["googleai_field"]
	assert.Equal(t, "googleai/gemini-1.5-pro", googleAIField.model, "Expected model 'googleai/gemini-1.5-pro'")

	// Check that plain model name is preserved
	plainField := sch.json2field["plain_field"]
	assert.Equal(t, "gemini-1.5-pro", plainField.model, "Expected model 'gemini-1.5-pro'")

	// Check that unknown/custom model name is preserved
	unknownField := sch.json2field["unknown_field"]
	assert.Equal(t, "custom-model-name", unknownField.model, "Expected model 'custom-model-name'")

	// Check that flash model is preserved
	flashField := sch.json2field["flash_field"]
	assert.Equal(t, "gemini-1.5-flash", flashField.model, "Expected model 'gemini-1.5-flash'")

	// Check prompt grouping with different models
	expectedGroups := []promptKey{
		{prompt: "provider_test", parentPath: "", model: "googleai/gemini-1.5-pro"},
		{prompt: "plain_test", parentPath: "", model: "gemini-1.5-pro"},
		{prompt: "unknown_test", parentPath: "", model: "custom-model-name"},
		{prompt: "flash_test", parentPath: "", model: "gemini-1.5-flash"},
	}

	for _, expectedKey := range expectedGroups {
		_, exists := sch.group2keys[expectedKey]
		assert.True(t, exists, "Expected prompt group %+v to exist", expectedKey)
	}
}

func TestNewSyntaxFormats(t *testing.T) {
	sch, err := schemaOf[NewSyntaxStruct]()
	require.NoError(t, err)

	testCases := []struct {
		field          string
		expectedPrompt string
		expectedModel  string
		description    string
	}{
		{
			field:          "explicit",
			expectedPrompt: "my_prompt",
			expectedModel:  "my_model",
			description:    "explicit prompt,model format should work",
		},
		{
			field:          "model_only",
			expectedPrompt: "",
			expectedModel:  "googleai/gemini-1.5-pro",
			description:    "model/ prefix should set model and inherit prompt",
		},
		{
			field:          "prompt_only",
			expectedPrompt: "custom_extraction",
			expectedModel:  "",
			description:    "prompt/ prefix should set prompt with no model",
		},
		{
			field:          "prompt_field",
			expectedPrompt: "regular_prompt",
			expectedModel:  "",
			description:    "regular prompt should work as before",
		},
		{
			field:          "provider_field",
			expectedPrompt: "extraction",
			expectedModel:  "vertex/gemini-1.5-flash",
			description:    "provider-prefixed models with explicit prompt should work",
		},
		{
			field:          "complex_model",
			expectedPrompt: "",
			expectedModel:  "anthropic/claude-3-sonnet",
			description:    "complex model names with model/ prefix should work",
		},
		{
			field:          "inherit_field",
			expectedPrompt: "",
			expectedModel:  "",
			description:    "empty tag should inherit from parent (empty in this case)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fieldSpec, exists := sch.json2field[tc.field]
			assert.True(t, exists, "Field %s should exist in schema", tc.field)

			assert.Equal(t, tc.expectedModel, fieldSpec.model, "Field %s: Expected model '%s', got '%s'", tc.field, tc.expectedModel, fieldSpec.model)

			// Check that the field is in the correct prompt group
			found := false
			for pk, fields := range sch.group2keys {
				if pk.prompt == tc.expectedPrompt && pk.model == tc.expectedModel {
					for _, field := range fields {
						if field == tc.field {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}
			assert.True(t, found, "Field %s not found in expected prompt group (prompt='%s', model='%s')",
				tc.field, tc.expectedPrompt, tc.expectedModel)
		})
	}
}

func TestTagParsing_NewFormats(t *testing.T) {
	// Test direct tag parsing with the new parseUnstructTag function
	testCases := []struct {
		tag             string
		inheritedPrompt string
		expectedPrompt  string
		expectedModel   string
		description     string
	}{
		{
			tag:             "model/openai/gpt-4",
			inheritedPrompt: "inherited",
			expectedPrompt:  "inherited",
			expectedModel:   "openai/gpt-4",
			description:     "model/ prefix should inherit prompt and set model",
		},
		{
			tag:             "prompt/my_extraction",
			inheritedPrompt: "",
			expectedPrompt:  "my_extraction",
			expectedModel:   "",
			description:     "prompt/ prefix should set prompt and no model",
		},
		{
			tag:             "custom_prompt,custom_model",
			inheritedPrompt: "",
			expectedPrompt:  "custom_prompt",
			expectedModel:   "custom_model",
			description:     "explicit format should work",
		},
		{
			tag:             "just_prompt",
			inheritedPrompt: "",
			expectedPrompt:  "just_prompt",
			expectedModel:   "",
			description:     "single value should be treated as prompt",
		},
		{
			tag:             "",
			inheritedPrompt: "parent_prompt",
			expectedPrompt:  "parent_prompt",
			expectedModel:   "",
			description:     "empty tag should inherit prompt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := parseUnstructTag(tc.tag, tc.inheritedPrompt, nil)
			assert.Equal(t, tc.expectedPrompt, result.prompt, "Tag %q: Expected prompt '%s', got '%s'", tc.tag, tc.expectedPrompt, result.prompt)
			assert.Equal(t, tc.expectedModel, result.model, "Tag %q: Expected model '%s', got '%s'", tc.tag, tc.expectedModel, result.model)
		})
	}
}

// Test that demonstrates the fix for provider-prefixed model issue
type ProviderPrefixIssueStruct struct {
	GoogleAIModel   string `json:"googleai_model" unstruct:"model/googleai/gemini-1.5-pro"`
	VertexModel     string `json:"vertex_model" unstruct:"model/vertex/gemini-1.5-flash"`
	BarePlainModel  string `json:"bare_model" unstruct:"model/gemini-1.5-pro"`
	UnknownProvider string `json:"unknown_provider" unstruct:"openai/gpt-4"` // This stays as prompt
}

func TestProviderPrefixModelRecognition(t *testing.T) {
	sch, err := schemaOf[ProviderPrefixIssueStruct]()
	require.NoError(t, err)

	testCases := []struct {
		field         string
		expectedModel string
		shouldBeModel bool
		description   string
	}{
		{
			field:         "googleai_model",
			expectedModel: "googleai/gemini-1.5-pro",
			shouldBeModel: true,
			description:   "googleai/ prefix should be recognized as model",
		},
		{
			field:         "vertex_model",
			expectedModel: "vertex/gemini-1.5-flash",
			shouldBeModel: true,
			description:   "vertex/ prefix should be recognized as model",
		},
		{
			field:         "bare_model",
			expectedModel: "gemini-1.5-pro",
			shouldBeModel: true,
			description:   "bare model name should work as before",
		},
		{
			field:         "unknown_provider",
			expectedModel: "",
			shouldBeModel: false,
			description:   "unknown provider should not be treated as model",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fieldSpec, exists := sch.json2field[tc.field]
			assert.True(t, exists, "Field %s should exist in schema", tc.field)

			if tc.shouldBeModel {
				assert.Equal(t, tc.expectedModel, fieldSpec.model, "Expected model '%s', got '%s'", tc.expectedModel, fieldSpec.model)
			} else {
				assert.Empty(t, fieldSpec.model, "Expected empty model (treated as prompt), got '%s'", fieldSpec.model)
			}
		})
	}
}

// BenchmarkSchemaOf tests performance of schema generation
func BenchmarkSchemaOf(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := schemaOf[SimpleStruct]()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Nested", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := schemaOf[NestedStruct]()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Complex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := schemaOf[ModelOverrideStruct]()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ProviderPrefix", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := schemaOf[ProviderPrefixStruct]()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Comprehensive test for all new syntax features
type ComprehensiveSyntaxTest struct {
	// Explicit prompt,model format
	ExplicitField string `json:"explicit" unstruct:"extract_title,googleai/gemini-1.5-pro"`

	// Model-only with various providers
	GoogleAIField  string `json:"googleai" unstruct:"model/googleai/gemini-1.5-flash"`
	VertexField    string `json:"vertex" unstruct:"model/vertex/gemini-1.5-pro"`
	AnthropicField string `json:"anthropic" unstruct:"model/anthropic/claude-3-sonnet"`
	OpenAIField    string `json:"openai" unstruct:"model/openai/gpt-4"`
	CustomField    string `json:"custom" unstruct:"model/custom-provider/my-model-v2"`

	// Prompt-only formats
	PromptPrefixField string `json:"prompt_prefix" unstruct:"prompt/special_extraction"`
	PromptOnlyField   string `json:"prompt_only" unstruct:"basic_prompt"`

	// Nested structure with inheritance
	NestedStruct struct {
		InheritedField1 string `json:"inherited1"`                                // Inherits everything
		InheritedField2 string `json:"inherited2"`                                // Inherits everything
		OverridePrompt  string `json:"override_prompt" unstruct:"new_prompt"`     // Override prompt only
		OverrideModel   string `json:"override_model" unstruct:"model/new/model"` // Override model only
	} `json:"nested" unstruct:"nested_extraction,vertex/gemini-1.5-flash"`

	// Empty tag (full inheritance)
	EmptyTagField string `json:"empty_tag" unstruct:""`
}

func TestComprehensiveSyntax(t *testing.T) {
	sch, err := schemaOf[ComprehensiveSyntaxTest]()
	require.NoError(t, err)

	testCases := []struct {
		field          string
		expectedPrompt string
		expectedModel  string
		description    string
	}{
		// Direct field tests
		{"explicit", "extract_title", "googleai/gemini-1.5-pro", "explicit format"},
		{"googleai", "", "googleai/gemini-1.5-flash", "googleai model"},
		{"vertex", "", "vertex/gemini-1.5-pro", "vertex model"},
		{"anthropic", "", "anthropic/claude-3-sonnet", "anthropic model"},
		{"openai", "", "openai/gpt-4", "openai model"},
		{"custom", "", "custom-provider/my-model-v2", "custom model"},
		{"prompt_prefix", "special_extraction", "", "prompt/ prefix"},
		{"prompt_only", "basic_prompt", "", "prompt only"},
		{"empty_tag", "", "", "empty tag inheritance"},

		// Nested field tests
		{"nested.inherited1", "nested_extraction", "vertex/gemini-1.5-flash", "nested inheritance"},
		{"nested.inherited2", "nested_extraction", "vertex/gemini-1.5-flash", "nested inheritance"},
		{"nested.override_prompt", "new_prompt", "vertex/gemini-1.5-flash", "override prompt, inherit model"},
		{"nested.override_model", "nested_extraction", "new/model", "inherit prompt, override model"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fieldSpec, exists := sch.json2field[tc.field]
			assert.True(t, exists, "Field %s should exist in schema", tc.field)

			assert.Equal(t, tc.expectedModel, fieldSpec.model, "Field %s: Expected model '%s', got '%s'", tc.field, tc.expectedModel, fieldSpec.model)

			// Verify the field is in the correct prompt group
			found := false
			for pk, fields := range sch.group2keys {
				if pk.prompt == tc.expectedPrompt && pk.model == tc.expectedModel {
					for _, field := range fields {
						if field == tc.field {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}
			assert.True(t, found, "Field %s not found in expected prompt group (prompt='%s', model='%s')",
				tc.field, tc.expectedPrompt, tc.expectedModel)
		})
	} // Verify we have the expected number of distinct groups
	expectedGroupCount := 12 // Count distinct (prompt, model) combinations
	assert.Len(t, sch.group2keys, expectedGroupCount, "Expected %d prompt groups", expectedGroupCount)
	if len(sch.group2keys) != expectedGroupCount {
		for pk, fields := range sch.group2keys {
			t.Logf("Group: prompt='%s', model='%s', parentPath='%s', fields=%v",
				pk.prompt, pk.model, pk.parentPath, fields)
		}
	}
}

// Test for nested struct field mapping issue
type NestedProject struct {
	ProjectCode string `json:"ProjectCode" unstruct:"base"`
	BSNumber    string `json:"BSNumber" unstruct:"base"`
	BSName      string `json:"BSName" unstruct:"base"`
	Standards   string `json:"Standards" unstruct:"base"`
}

type NestedMeta struct {
	ParserVersion string      `json:"ParserVersion" unstruct:"base"`
	Timestamp     string      `json:"Timestamp" unstruct:"base"`
	DocsAnalysed  interface{} `json:"DocsAnalysed" unstruct:"base"`
}

type AerialsStruct struct {
	Project NestedProject `json:"Project" unstruct:"base"`
	Meta    NestedMeta    `json:"Meta" unstruct:"base"`
}

func TestNestedStructFieldMappingFix(t *testing.T) {
	t.Run("Schema Generation for Nested Structs", func(t *testing.T) {
		sch, err := schemaOf[AerialsStruct]()
		require.NoError(t, err)

		// Check if nested struct fields are in the schema
		expectedFields := []string{
			"Project.ProjectCode",
			"Project.BSNumber",
			"Project.BSName",
			"Project.Standards",
			"Meta.ParserVersion",
			"Meta.Timestamp",
			"Meta.DocsAnalysed",
		}

		for _, field := range expectedFields {
			_, exists := sch.json2field[field]
			assert.True(t, exists, "Expected field %s to exist in schema", field)
		}

		// Check if intermediate struct fields are in the schema (this should be fixed now)
		intermediateFields := []string{"Project", "Meta"}
		for _, field := range intermediateFields {
			_, exists := sch.json2field[field]
			if !exists {
				t.Errorf("Intermediate field %s missing from schema", field)
			} else {
				t.Logf("Intermediate field %s now exists in schema", field)
			}
		}

		// Print schema for debugging
		t.Logf("Schema fields: %+v", sch.json2field)
	})

	t.Run("JSON Patching with Nested Structure", func(t *testing.T) {
		// This is the nested JSON that AI returns
		nestedJSON := `{
			"Project": {
				"ProjectCode": "TEST-001",
				"BSNumber": "BS-123", 
				"BSName": "Test Base Station",
				"Standards": "LTE-1800"
			},
			"Meta": {
				"ParserVersion": "1.0.0",
				"Timestamp": "2025-07-14T10:00:00Z",
				"DocsAnalysed": {"doc1": "analyzed"}
			}
		}`

		sch, err := schemaOf[AerialsStruct]()
		require.NoError(t, err)

		var result AerialsStruct
		err = patchStruct(&result, []byte(nestedJSON), sch.json2field)
		require.NoError(t, err)

		// Check if nested fields are populated (should be fixed now)
		assert.Equal(t, "TEST-001", result.Project.ProjectCode, "Expected Project.ProjectCode 'TEST-001'")
		t.Logf("Project.ProjectCode correctly set to '%s'", result.Project.ProjectCode)

		assert.Equal(t, "1.0.0", result.Meta.ParserVersion, "Expected Meta.ParserVersion '1.0.0'")
		t.Logf("Meta.ParserVersion correctly set to '%s'", result.Meta.ParserVersion)

		t.Logf("Final Result: %+v", result)
	})
}
