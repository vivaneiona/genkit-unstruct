package unstruct

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

const unstractTag = "unstruct"

type promptKey struct {
	prompt     string // explicit label or ""
	parentPath string // dotted path w/o the leaf field
	model      string // model name for this group
}

type fieldSpec struct {
	jsonKey string
	model   string // may be ""
	index   []int  // reflect path
}

type schema struct {
	group2keys map[promptKey][]string // batching groups
	json2field map[string]fieldSpec   // merge map
}

func schemaOf[T any]() (*schema, error) {
	return schemaOfWithOptions[T](nil)
}

func schemaOfWithOptions[T any](opts *Options) (*schema, error) {
	var zero T
	rt := reflect.TypeOf(zero)
	if rt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("unstruct: T must be struct")
	}

	s := &schema{
		group2keys: map[promptKey][]string{},
		json2field: map[string]fieldSpec{},
	}

	var walk func(t reflect.Type, parent, inheritedPrompt, inheritedModel string, idx []int)
	walk = func(t reflect.Type, parent, inheritedPrompt, inheritedModel string, idx []int) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous || !f.IsExported() {
				continue
			}
			jsonKey := strings.Split(f.Tag.Get("json"), ",")[0]
			if jsonKey == "-" {
				continue
			}
			if jsonKey == "" {
				jsonKey = f.Name
			}
			fullKey := joinKey(parent, jsonKey)
			tp := parseUnstructTag(f.Tag.Get("unstruct"), inheritedPrompt)

			// Resolve group references
			prompt := tp.prompt
			model := tp.model
			if strings.HasPrefix(tp.prompt, "group:") {
				groupName := strings.TrimPrefix(tp.prompt, "group:")
				if opts != nil && opts.Groups != nil {
					if groupDef, exists := opts.Groups[groupName]; exists {
						prompt = groupDef.Prompt
						model = groupDef.Model
					}
				}
			}

			// Inherit model if not specified in the tag and not from group
			if model == "" {
				model = inheritedModel
			}

			// Check for field-specific model override from Options
			if opts != nil && opts.FieldModels != nil {
				typeName := t.Name()
				fieldKey := typeName + "." + f.Name
				if fieldModel, exists := opts.FieldModels[fieldKey]; exists {
					model = fieldModel
				}
			}

			nextIdx := append(idx, i)
			if isPureStruct(f.Type) {
				// Make the intermediate node addressable during patching
				s.json2field[fullKey] = fieldSpec{
					jsonKey: fullKey,
					model:   model,
					index:   nextIdx,
				}
				walk(f.Type, fullKey, prompt, model, nextIdx)
				continue
			}

			// Handle slices of structs
			if f.Type.Kind() == reflect.Slice && isPureStruct(f.Type.Elem()) {
				s.json2field[fullKey] = fieldSpec{
					jsonKey: fullKey,
					model:   model,
					index:   nextIdx,
				}
				walk(f.Type.Elem(), fullKey, prompt, model, nextIdx)
				continue
			}

			// Create prompt key, optionally flattening groups
			parentPathForGrouping := parent
			if opts != nil && opts.FlattenGroups {
				parentPathForGrouping = ""
			}

			pk := promptKey{prompt: prompt, parentPath: parentPathForGrouping, model: model}
			s.group2keys[pk] = append(s.group2keys[pk], fullKey)
			s.json2field[fullKey] = fieldSpec{
				jsonKey: fullKey, model: model, index: nextIdx,
			}
		}
	}
	walk(rt, "", "", "", nil)
	return s, nil
}

// helpers
func joinKey(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func isPureStruct(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && t != reflect.TypeOf(time.Time{})
}
