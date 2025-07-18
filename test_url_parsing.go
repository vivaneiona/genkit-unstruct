package unstruct

import (
	"fmt"
	"testing"
)

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
			fmt.Printf("Tag: %s\n", test.tag)
			fmt.Printf("Result: prompt='%s', model='%s', params=%v\n",
				result.prompt, result.model, result.parameters)

			if result.prompt != test.expectedPrompt {
				t.Errorf("Expected prompt '%s', got '%s'", test.expectedPrompt, result.prompt)
			}
			if result.model != test.expectedModel {
				t.Errorf("Expected model '%s', got '%s'", test.expectedModel, result.model)
			}

			for _, key := range test.expectedParamKeys {
				if _, exists := result.parameters[key]; !exists {
					t.Errorf("Expected parameter '%s' to exist", key)
				}
			}
		})
	}
}
