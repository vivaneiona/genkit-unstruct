package unstruct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewURLStyleSyntax(t *testing.T) {
	testCases := []struct {
		tag             string
		inheritedPrompt string
		expectedPrompt  string
		expectedModel   string
		description     string
	}{
		{
			tag:             "prompt/promptname/model/gemini-1.5-pro",
			inheritedPrompt: "",
			expectedPrompt:  "promptname",
			expectedModel:   "gemini-1.5-pro",
			description:     "URL-style syntax with prompt and model",
		},
		{
			tag:             "model/gemini-2.0-flash",
			inheritedPrompt: "inherited",
			expectedPrompt:  "inherited",
			expectedModel:   "gemini-2.0-flash",
			description:     "Simple model/ prefix (updated syntax)",
		},
		{
			tag:             "prompt,gemini-1.5-pro",
			inheritedPrompt: "",
			expectedPrompt:  "prompt",
			expectedModel:   "gemini-1.5-pro",
			description:     "Legacy comma syntax still works",
		},
		{
			tag:             "model/openai/gpt-4",
			inheritedPrompt: "inherited",
			expectedPrompt:  "inherited",
			expectedModel:   "openai/gpt-4",
			description:     "Complex model name with provider prefix",
		},
		{
			tag:             "model/anthropic/claude-3-sonnet",
			inheritedPrompt: "inherited",
			expectedPrompt:  "inherited",
			expectedModel:   "anthropic/claude-3-sonnet",
			description:     "Another complex model name",
		},
		{
			tag:             "prompt/extraction/model/vertex/gemini-1.5-flash",
			inheritedPrompt: "",
			expectedPrompt:  "extraction",
			expectedModel:   "vertex/gemini-1.5-flash",
			description:     "URL-style with complex model name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := parseUnstructTag(tc.tag, tc.inheritedPrompt)
			assert.Equal(t, tc.expectedPrompt, result.prompt, "Tag %q: prompt mismatch", tc.tag)
			assert.Equal(t, tc.expectedModel, result.model, "Tag %q: model mismatch", tc.tag)
		})
	}
}

func TestURLParsingWithParameters(t *testing.T) {
	testCases := []struct {
		tag               string
		expectedPrompt    string
		expectedModel     string
		expectedParamKeys []string
	}{
		{
			tag:               "model/vertex/gemini-1.5-flash?temperature=0.5&topK=10",
			expectedPrompt:    "inherited",
			expectedModel:     "vertex/gemini-1.5-flash",
			expectedParamKeys: []string{"temperature", "topK"},
		},
		{
			tag:               "prompt/extraction/model/vertex/gemini-1.5-flash?temperature=0",
			expectedPrompt:    "extraction",
			expectedModel:     "vertex/gemini-1.5-flash",
			expectedParamKeys: []string{"temperature"},
		},
		{
			tag:               "model/openai/gpt-4?temperature=0.7&maxTokens=1000",
			expectedPrompt:    "inherited",
			expectedModel:     "openai/gpt-4",
			expectedParamKeys: []string{"temperature", "maxTokens"},
		},
		{
			tag:               "prompt/detailed?temperature=0.3",
			expectedPrompt:    "detailed",
			expectedModel:     "",
			expectedParamKeys: []string{"temperature"},
		},
	}

	for _, test := range testCases {
		t.Run(test.tag, func(t *testing.T) {
			result := parseUnstructTag(test.tag, "inherited")

			assert.Equal(t, test.expectedPrompt, result.prompt, "Expected prompt '%s', got '%s'", test.expectedPrompt, result.prompt)
			assert.Equal(t, test.expectedModel, result.model, "Expected model '%s', got '%s'", test.expectedModel, result.model)

			for _, key := range test.expectedParamKeys {
				assert.Contains(t, result.parameters, key, "Expected parameter '%s' to exist", key)
			}
		})
	}
}

func TestQueryParameterParsing(t *testing.T) {
	// Test tag with query parameters
	tag := "model/vertex/gemini-1.5-flash?temperature=0.5&topK=10"
	result := parseUnstructTag(tag, "inherited")

	// Check basic parsing
	assert.Equal(t, "inherited", result.prompt, "Expected prompt 'inherited', got '%s'", result.prompt)
	assert.Equal(t, "vertex/gemini-1.5-flash", result.model, "Expected model 'vertex/gemini-1.5-flash', got '%s'", result.model)

	// Check parameters
	require.NotNil(t, result.parameters, "Expected parameters map to be initialized")

	temp, exists := result.parameters["temperature"]
	assert.True(t, exists, "Expected 'temperature' parameter to exist")
	assert.Equal(t, "0.5", temp, "Expected temperature '0.5', got '%s'", temp)

	topK, exists := result.parameters["topK"]
	assert.True(t, exists, "Expected 'topK' parameter to exist")
	assert.Equal(t, "10", topK, "Expected topK '10', got '%s'", topK)
}
