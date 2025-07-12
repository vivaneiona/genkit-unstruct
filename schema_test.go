package unstruct

import (
	"reflect"
	"testing"
	"time"
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
	BasicField  string `json:"basic" unstruct:"basic"`
	FlashField  string `json:"flash" unstruct:"fast,gemini-1.5-flash"`
	ProField    string `json:"pro" unstruct:"complex,gemini-1.5-pro"`
	ModelOnly   string `json:"model_only" unstruct:"gemini-1.5-flash-8b"`
}

type InheritanceStruct struct {
	Parent ParentStruct `json:"parent" unstruct:"inherited_prompt"`
}

type ParentStruct struct {
	Child1 string `json:"child1"` // Should inherit "inherited_prompt"
	Child2 string `json:"child2" unstruct:"override"` // Should override
}

type EdgeCaseStruct struct {
	Ignored       string    `json:"-"`
	NoTag         string    `json:"no_tag"`
	EmptyTag      string    `json:"empty_tag" unstruct:""`
	TimeField     time.Time `json:"time_field" unstruct:"temporal"`
	unexported    string    `json:"unexported" unstruct:"basic"`
	Anonymous     struct{ Field string } // Anonymous struct
}

func TestSchemaOf_SimpleStruct(t *testing.T) {
	sch, err := schemaOf[SimpleStruct]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Should have one prompt group
	if len(sch.group2keys) != 1 {
		t.Errorf("Expected 1 prompt group, got %d", len(sch.group2keys))
	}

	// Should have two fields
	if len(sch.json2field) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(sch.json2field))
	}

	// Check the basic group
	basicKey := promptKey{prompt: "basic", parentPath: ""}
	keys, exists := sch.group2keys[basicKey]
	if !exists {
		t.Error("Expected 'basic' prompt group to exist")
	}
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys in basic group, got %d", len(keys))
	}

	// Check field specs
	nameField, exists := sch.json2field["name"]
	if !exists {
		t.Error("Expected 'name' field to exist")
	}
	if nameField.jsonKey != "name" {
		t.Errorf("Expected jsonKey 'name', got %s", nameField.jsonKey)
	}
	if len(nameField.index) != 1 {
		t.Errorf("Expected index length 1, got %d", len(nameField.index))
	}
}

func TestSchemaOf_NestedStruct(t *testing.T) {
	sch, err := schemaOf[NestedStruct]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Should have multiple prompt groups
	expectedGroups := []promptKey{
		{prompt: "name", parentPath: "user"},
		{prompt: "contact", parentPath: "user"},
		{prompt: "project", parentPath: "project"},
	}

	for _, expectedKey := range expectedGroups {
		if _, exists := sch.group2keys[expectedKey]; !exists {
			t.Errorf("Expected prompt group %+v to exist", expectedKey)
		}
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
		if _, exists := sch.json2field[field]; !exists {
			t.Errorf("Expected field %s to exist", field)
		}
	}
}

func TestSchemaOf_SliceStruct(t *testing.T) {
	sch, err := schemaOf[SliceStruct]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Check that slice of structs is handled correctly
	expectedFields := []string{
		"tags",
		"users.first_name",
		"users.last_name",
		"users.email",
		"numbers",
	}

	for _, field := range expectedFields {
		if _, exists := sch.json2field[field]; !exists {
			t.Errorf("Expected field %s to exist", field)
		}
	}
}

func TestSchemaOf_ModelOverride(t *testing.T) {
	sch, err := schemaOf[ModelOverrideStruct]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Check model specifications
	flashField := sch.json2field["flash"]
	if flashField.model != "gemini-1.5-flash" {
		t.Errorf("Expected model 'gemini-1.5-flash', got %s", flashField.model)
	}

	proField := sch.json2field["pro"]
	if proField.model != "gemini-1.5-pro" {
		t.Errorf("Expected model 'gemini-1.5-pro', got %s", proField.model)
	}

	modelOnlyField := sch.json2field["model_only"]
	if modelOnlyField.model != "gemini-1.5-flash-8b" {
		t.Errorf("Expected model 'gemini-1.5-flash-8b', got %s", modelOnlyField.model)
	}

	basicField := sch.json2field["basic"]
	if basicField.model != "" {
		t.Errorf("Expected empty model for basic field, got %s", basicField.model)
	}
}

func TestSchemaOf_Inheritance(t *testing.T) {
	sch, err := schemaOf[InheritanceStruct]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Check that child1 inherits the prompt
	inheritedKey := promptKey{prompt: "inherited_prompt", parentPath: "parent"}
	keys, exists := sch.group2keys[inheritedKey]
	if !exists {
		t.Error("Expected inherited prompt group to exist")
	}
	
	found := false
	for _, key := range keys {
		if key == "parent.child1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected parent.child1 to inherit prompt")
	}

	// Check that child2 overrides the prompt
	overrideKey := promptKey{prompt: "override", parentPath: "parent"}
	keys, exists = sch.group2keys[overrideKey]
	if !exists {
		t.Error("Expected override prompt group to exist")
	}
	
	found = false
	for _, key := range keys {
		if key == "parent.child2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected parent.child2 to override prompt")
	}
}

func TestSchemaOf_EdgeCases(t *testing.T) {
	sch, err := schemaOf[EdgeCaseStruct]()
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Check ignored fields are not included
	if _, exists := sch.json2field["ignored"]; exists {
		t.Error("Ignored field should not exist in schema")
	}

	if _, exists := sch.json2field["unexported"]; exists {
		t.Error("Unexported field should not exist in schema")
	}

	// Check that fields without tags still get included
	if _, exists := sch.json2field["no_tag"]; !exists {
		t.Error("Field without tag should still exist")
	}

	// Check time field is treated as leaf (not struct)
	if _, exists := sch.json2field["time_field"]; !exists {
		t.Error("Time field should exist as leaf")
	}
}

func TestSchemaOf_NonStructType(t *testing.T) {
	// Test with non-struct type
	type StringType string
	_, err := schemaOf[StringType]()
	if err == nil {
		t.Error("Expected error for non-struct type")
	}
	if err != nil && err.Error() != "unstruct: T must be struct" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	// Test with slice type
	_, err = schemaOf[[]string]()
	if err == nil {
		t.Error("Expected error for slice type")
	}

	// Test with map type
	_, err = schemaOf[map[string]string]()
	if err == nil {
		t.Error("Expected error for map type")
	}
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
		if result != test.expected {
			t.Errorf("joinKey(%q, %q) = %q, expected %q", 
				test.parent, test.child, result, test.expected)
		}
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
		if result != test.expected {
			t.Errorf("isPureStruct(%v) = %v, expected %v", 
				test.typ, result, test.expected)
		}
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
	if err != nil {
		t.Fatalf("schemaOf failed: %v", err)
	}

	// Check group1 has 2 fields
	group1Key := promptKey{prompt: "group1", parentPath: ""}
	keys1, exists := sch.group2keys[group1Key]
	if !exists {
		t.Error("Expected group1 to exist")
	}
	if len(keys1) != 2 {
		t.Errorf("Expected 2 keys in group1, got %d", len(keys1))
	}

	// Check group2 has 2 fields
	group2Key := promptKey{prompt: "group2", parentPath: ""}
	keys2, exists := sch.group2keys[group2Key]
	if !exists {
		t.Error("Expected group2 to exist")
	}
	if len(keys2) != 2 {
		t.Errorf("Expected 2 keys in group2, got %d", len(keys2))
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
}
