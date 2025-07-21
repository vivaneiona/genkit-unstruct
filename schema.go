package unstruct

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// hashParameters creates a deterministic hash of parameters for grouping
func hashParameters(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	// Sort keys for deterministic hashing
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create a sorted string representation
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}

	hashStr := strings.Join(parts, "&")
	return fmt.Sprintf("%x", md5.Sum([]byte(hashStr)))[:8] // Use first 8 chars of hash
}

type promptKey struct {
	prompt     string // explicit label or ""
	parentPath string // dotted path w/o the leaf field
	model      string // model name for this group
	paramsHash string // hash of parameters for grouping
}

type promptGroup struct {
	promptKey
	parameters map[string]string // query parameters for this group
}

type fieldSpec struct {
	jsonKey    string
	model      string            // may be ""
	parameters map[string]string // query parameters for this field
	index      []int             // reflect path
}

type schema struct {
	group2keys  map[promptKey][]string    // batching groups (using comparable key)
	group2specs map[promptKey]promptGroup // stores the full group info with parameters
	json2field  map[string]fieldSpec      // merge map
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
		group2keys:  map[promptKey][]string{},
		group2specs: map[promptKey]promptGroup{},
		json2field:  map[string]fieldSpec{},
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
			parameters := tp.parameters
			if strings.HasPrefix(tp.prompt, "group:") {
				groupName := strings.TrimPrefix(tp.prompt, "group:")
				if opts != nil && opts.Groups != nil {
					if groupDef, exists := opts.Groups[groupName]; exists {
						prompt = groupDef.Prompt
						model = groupDef.Model
						// Group parameters could be merged here if needed
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
					jsonKey:    fullKey,
					model:      model,
					parameters: parameters,
					index:      nextIdx,
				}
				walk(f.Type, fullKey, prompt, model, nextIdx)
				continue
			}

			// Handle slices of structs
			if f.Type.Kind() == reflect.Slice && isPureStruct(f.Type.Elem()) {
				s.json2field[fullKey] = fieldSpec{
					jsonKey:    fullKey,
					model:      model,
					parameters: parameters,
					index:      nextIdx,
				}
				walk(f.Type.Elem(), fullKey, prompt, model, nextIdx)
				continue
			}

			// Create prompt key, optionally flattening groups
			parentPathForGrouping := parent
			if opts != nil && opts.FlattenGroups {
				parentPathForGrouping = ""
			}

			pk := promptKey{
				prompt:     prompt,
				parentPath: parentPathForGrouping,
				model:      model,
				paramsHash: hashParameters(parameters),
			}
			s.group2keys[pk] = append(s.group2keys[pk], fullKey)
			s.group2specs[pk] = promptGroup{
				promptKey:  pk,
				parameters: parameters,
			}
			s.json2field[fullKey] = fieldSpec{
				jsonKey:    fullKey,
				model:      model,
				parameters: parameters,
				index:      nextIdx,
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
