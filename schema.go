package unstruct

import (
	"log/slog"
	"reflect"
	"strings"
	"sync"
)

var schemaCache sync.Map

type cachedSchema struct {
	tag2keys   map[string][]string
	json2field map[string]reflect.StructField
}

const tagName = "extractor"

func schemaOf[T any]() (map[string][]string, map[string]reflect.StructField) {
	var zero T
	rt := reflect.TypeOf(zero)
	typeName := rt.String()
	slog.Debug("Analyzing type", "type", typeName, "kind", rt.Kind())

	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
		slog.Debug("Dereferenced pointer", "elem_type", rt.String())
	}

	if v, ok := schemaCache.Load(rt); ok {
		s := v.(cachedSchema)
		slog.Debug("Cache hit", "type", rt.String(), "tag_count", len(s.tag2keys), "field_count", len(s.json2field))
		return s.tag2keys, s.json2field
	}

	slog.Debug("Cache miss, analyzing struct", "type", rt.String())
	tag2keys := map[string][]string{}
	json2field := map[string]reflect.StructField{}

	fieldCount := rt.NumField()
	slog.Debug("Iterating fields", "field_count", fieldCount)

	for i := 0; i < fieldCount; i++ {
		f := rt.Field(i)
		jsonKey, _, _ := strings.Cut(f.Tag.Get("json"), ",") // strip modifiers
		if jsonKey == "" {
			slog.Debug("skipping field without json tag", "field_name", f.Name)
			continue
		}
		tag := f.Tag.Get(tagName)
		if tag == "" {
			tag = "default"
		}
		slog.Debug("Processed field", "field_name", f.Name, "json_key", jsonKey, "extractor_tag", tag)
		tag2keys[tag] = append(tag2keys[tag], jsonKey)
		json2field[jsonKey] = f
	}

	s := cachedSchema{tag2keys, json2field}
	schemaCache.Store(rt, s)
	slog.Debug("Cached schema", "type", rt.String(), "tag_count", len(s.tag2keys), "field_count", len(s.json2field))
	return s.tag2keys, s.json2field
}
