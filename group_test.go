package unstruct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithGroup(t *testing.T) {
	type PersonWithGroups struct {
		Name string `json:"name" unstruct:"group/basic-info"`
		Age  int    `json:"age"  unstruct:"group/basic-info"`
		City string `json:"city" unstruct:"group/basic-info"`
	}

	t.Run("schema analysis with groups", func(t *testing.T) {
		opts := &Options{}
		WithGroup("basic-info", "basic", "gemini-2.0-flash")(opts)

		schema, err := schemaOfWithOptions[PersonWithGroups](opts, nil)
		require.NoError(t, err)

		// Should have one group since all fields use the same group
		assert.Equal(t, 1, len(schema.group2keys))

		// Check that the group is properly resolved
		var foundGroup promptKey
		for pk := range schema.group2keys {
			foundGroup = pk
			break
		}

		assert.Equal(t, "basic", foundGroup.prompt)
		assert.Equal(t, "gemini-2.0-flash", foundGroup.model)

		// All three fields should be in the same group
		fields := schema.group2keys[foundGroup]
		assert.Len(t, fields, 3)
		assert.Contains(t, fields, "name")
		assert.Contains(t, fields, "age")
		assert.Contains(t, fields, "city")
	})

	t.Run("mixed group and direct tags", func(t *testing.T) {
		type MixedPerson struct {
			Name    string `json:"name" unstruct:"group/basic-info"`
			Age     int    `json:"age"  unstruct:"group/basic-info"`
			Address string `json:"address" unstruct:"address,gemini-1.5-pro"`
		}

		opts := &Options{}
		WithGroup("basic-info", "basic", "gemini-2.0-flash")(opts)

		schema, err := schemaOfWithOptions[MixedPerson](opts, nil)
		require.NoError(t, err)

		// Should have two groups
		assert.Equal(t, 2, len(schema.group2keys))

		// Check groups
		groups := make(map[string]promptKey)
		for pk := range schema.group2keys {
			switch pk.prompt {
			case "basic":
				groups["basic"] = pk
			case "address":
				groups["address"] = pk
			}
		}

		assert.Len(t, groups, 2)

		// Basic group should have name and age
		basicFields := schema.group2keys[groups["basic"]]
		assert.Len(t, basicFields, 2)
		assert.Contains(t, basicFields, "name")
		assert.Contains(t, basicFields, "age")

		// Address group should have address
		addressFields := schema.group2keys[groups["address"]]
		assert.Len(t, addressFields, 1)
		assert.Contains(t, addressFields, "address")
	})

	t.Run("undefined group reference", func(t *testing.T) {
		type PersonWithUndefinedGroup struct {
			Name string `json:"name" unstruct:"group/undefined-group"`
		}

		opts := &Options{}
		// Don't define the group

		schema, err := schemaOfWithOptions[PersonWithUndefinedGroup](opts, nil)
		require.NoError(t, err)

		// Should still create a group, but with the group marker as prompt
		assert.Equal(t, 1, len(schema.group2keys))

		var foundGroup promptKey
		for pk := range schema.group2keys {
			foundGroup = pk
			break
		}

		// Should keep the group marker since it wasn't resolved
		assert.Equal(t, "group:undefined-group", foundGroup.prompt)
	})
}

func TestGroupTagParsing(t *testing.T) {
	tests := []struct {
		name            string
		tag             string
		inheritedPrompt string
		expectedPrompt  string
		expectedModel   string
	}{
		{
			name:            "group reference",
			tag:             "group/basic-info",
			inheritedPrompt: "parent",
			expectedPrompt:  "group:basic-info",
			expectedModel:   "",
		},
		{
			name:            "regular prompt",
			tag:             "basic,gemini-flash",
			inheritedPrompt: "parent",
			expectedPrompt:  "basic",
			expectedModel:   "gemini-flash",
		},
		{
			name:            "model only",
			tag:             "model/gemini-pro",
			inheritedPrompt: "parent",
			expectedPrompt:  "parent",
			expectedModel:   "gemini-pro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUnstructTag(tt.tag, tt.inheritedPrompt, nil)
			assert.Equal(t, tt.expectedPrompt, result.prompt)
			assert.Equal(t, tt.expectedModel, result.model)
		})
	}
}
