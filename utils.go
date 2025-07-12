package unstruct

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
)

// buildPrompt constructs the final prompt from template, keys, and document
func buildPrompt(tpl string, keys []string, doc string) string {
	slog.Debug("starting prompt construction", "template_length", len(tpl), "keys_count", len(keys), "document_length", len(doc))

	originalTpl := tpl
	if strings.Contains(tpl, "{{.Keys}}") {
		keysStr := strings.Join(keys, ",")
		tpl = strings.ReplaceAll(tpl, "{{.Keys}}", keysStr)
		slog.Debug("Replaced keys placeholder", "keys", keysStr)
	}

	finalPrompt := tpl + "\n\n<<DOC>>\n" + doc + "\n<<END>>"
	slog.Debug("constructed final prompt", "original_template_length", len(originalTpl), "final_prompt_length", len(finalPrompt))

	return finalPrompt
}

// patchStruct merges JSON data into the destination struct using field mapping
func patchStruct[T any](
	dst *T,
	raw []byte,
	fieldDict map[string]reflect.StructField,
) error {
	slog.Debug("Starting JSON merge", "raw_length", len(raw), "field_dict_size", len(fieldDict), "raw_preview", string(raw)[:min(200, len(raw))])

	sanitized := SanitizeJSONResponse(raw)
	slog.Debug("Sanitized JSON", "sanitized_length", len(sanitized))

	var kv map[string]json.RawMessage
	if err := json.Unmarshal(sanitized, &kv); err != nil {
		slog.Debug("JSON unmarshal failed", "error", err, "sanitized_json", string(sanitized))
		return err
	}
	slog.Debug("Unmarshaled JSON", "key_count", len(kv))

	rv := reflect.ValueOf(dst).Elem()
	patchedFields := 0
	for key, b := range kv {
		slog.Debug("Processing key", "key", key, "value_length", len(b))
		f, ok := fieldDict[key]
		if !ok || len(b) == 0 {
			slog.Debug("Skipping key", "key", key, "field_found", ok, "value_empty", len(b) == 0)
			continue
		}
		fv := rv.FieldByIndex(f.Index)
		if !fv.CanSet() {
			slog.Debug("Field not settable", "key", key, "field_name", f.Name)
			continue
		}
		ptr := reflect.New(f.Type)
		if err := json.Unmarshal(b, ptr.Interface()); err != nil {
			slog.Debug("Field unmarshal failed", "key", key, "field_name", f.Name, "error", err)
			return fmt.Errorf("field %s: %w", key, err)
		}
		fv.Set(ptr.Elem())
		patchedFields++
		slog.Debug("Patched field", "key", key, "field_name", f.Name)
	}
	slog.Debug("Completed JSON merge", "patched_fields", patchedFields)
	return nil
}

// SanitizeJSONResponse removes garbage characters often produced by LLMs.
// Very defensive, yet fast; tweak as you like.
func SanitizeJSONResponse(b []byte) []byte {
	slog.Debug("Starting sanitization", "input_length", len(b), "input_preview", string(b)[:min(100, len(b))])

	s := strings.TrimSpace(string(b))
	slog.Debug("Trimmed whitespace", "length_after_trim", len(s))

	// Remove leading/trailing code fences, markdown, etc.
	originalLen := len(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	finalS := strings.TrimSpace(s)

	slog.Debug("Sanitization complete", "original_length", originalLen, "final_length", len(finalS), "removed_prefixes_suffixes", originalLen != len(finalS))

	return []byte(finalS)
}
