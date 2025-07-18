package unstruct

import (
	"testing"
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
			if result.prompt != tc.expectedPrompt {
				t.Errorf("Tag %q: Expected prompt '%s', got '%s'", tc.tag, tc.expectedPrompt, result.prompt)
			}
			if result.model != tc.expectedModel {
				t.Errorf("Tag %q: Expected model '%s', got '%s'", tc.tag, tc.expectedModel, result.model)
			}
		})
	}
}
