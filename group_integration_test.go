package unstruct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupIntegration(t *testing.T) {
	type Company struct {
		Name    string `json:"name" unstruct:"group/company-info"`
		Address string `json:"address" unstruct:"group/company-info"`
		Phone   string `json:"phone" unstruct:"group/contact"`
		Email   string `json:"email" unstruct:"group/contact"`
	}

	type Employee struct {
		Name    string  `json:"name" unstruct:"group/personal"`
		Age     int     `json:"age" unstruct:"group/personal"`
		Company Company `json:"company"`
		Salary  float64 `json:"salary" unstruct:"financial,gemini-1.5-pro"`
	}

	t.Run("nested structures with groups", func(t *testing.T) {
		opts := &Options{}
		WithGroup("company-info", "company", "gemini-2.0-flash")(opts)
		WithGroup("contact", "contact", "gemini-1.5-flash")(opts)
		WithGroup("personal", "personal", "gemini-2.0-flash")(opts)

		schema, err := schemaOfWithOptions[Employee](opts)
		require.NoError(t, err)

		// Should have 4 groups:
		// 1. personal (name, age)
		// 2. company (company.name, company.address)
		// 3. contact (company.phone, company.email)
		// 4. financial (salary)
		assert.Equal(t, 4, len(schema.group2keys))

		// Verify each group has the correct fields
		groupFieldCounts := make(map[string]int)
		for pk, fields := range schema.group2keys {
			groupFieldCounts[pk.prompt] = len(fields)
		}

		assert.Equal(t, 2, groupFieldCounts["personal"])
		assert.Equal(t, 2, groupFieldCounts["company"])
		assert.Equal(t, 2, groupFieldCounts["contact"])
		assert.Equal(t, 1, groupFieldCounts["financial"])
	})

	t.Run("group inheritance in nested structures", func(t *testing.T) {
		type NestedStruct struct {
			Field1 string `json:"field1" unstruct:"group/inherited"`
			Field2 string `json:"field2"` // should inherit from parent
		}

		type ParentStruct struct {
			Nested NestedStruct `json:"nested" unstruct:"group/parent"`
		}

		opts := &Options{}
		WithGroup("inherited", "inherited-prompt", "gemini-1.5-flash")(opts)
		WithGroup("parent", "parent-prompt", "gemini-2.0-flash")(opts)

		schema, err := schemaOfWithOptions[ParentStruct](opts)
		require.NoError(t, err)

		// Should have 2 groups
		assert.Equal(t, 2, len(schema.group2keys))

		// Find the groups
		var inheritedGroup, parentGroup promptKey
		for pk := range schema.group2keys {
			if pk.prompt == "inherited-prompt" {
				inheritedGroup = pk
			} else if pk.prompt == "parent-prompt" {
				parentGroup = pk
			}
		}

		// field1 should use the inherited group
		assert.Contains(t, schema.group2keys[inheritedGroup], "nested.field1")
		// field2 should inherit from parent
		assert.Contains(t, schema.group2keys[parentGroup], "nested.field2")
	})
}

func TestGroupWithFlattenGroups(t *testing.T) {
	type Level1 struct {
		Field1 string `json:"field1" unstruct:"group/shared"`
	}

	type Level2 struct {
		Field2 string `json:"field2" unstruct:"group/shared"`
		Level1 Level1 `json:"level1"`
	}

	type Root struct {
		Field3 string `json:"field3" unstruct:"group/shared"`
		Level2 Level2 `json:"level2"`
	}

	t.Run("without flattening", func(t *testing.T) {
		opts := &Options{}
		WithGroup("shared", "shared-prompt", "gemini-2.0-flash")(opts)

		schema, err := schemaOfWithOptions[Root](opts)
		require.NoError(t, err)

		// Without flattening, different parent paths create separate groups
		assert.True(t, len(schema.group2keys) > 1)
	})

	t.Run("with flattening", func(t *testing.T) {
		opts := &Options{}
		WithGroup("shared", "shared-prompt", "gemini-2.0-flash")(opts)
		WithFlattenGroups()(opts)

		schema, err := schemaOfWithOptions[Root](opts)
		require.NoError(t, err)

		// With flattening, should have only 1 group since all fields use the same prompt+model
		assert.Equal(t, 1, len(schema.group2keys))

		// All fields should be in the same group
		var allFields []string
		for _, fields := range schema.group2keys {
			allFields = fields
			break
		}

		assert.Len(t, allFields, 3)
		assert.Contains(t, allFields, "field3")
		assert.Contains(t, allFields, "level2.field2")
		assert.Contains(t, allFields, "level2.level1.field1")
	})
}
