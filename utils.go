package unstruct

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"
)

// traverse returns value and a boolean 'found'.
func traverse(m map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	cur := any(m)
	for _, p := range parts {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = mm[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// patchStruct merges JSON data into the destination struct using dotted keys support
func patchStruct[T any](dst *T, raw []byte, spec map[string]fieldSpec) error {
	slog.Debug("Starting JSON merge", "raw_length", len(raw), "spec_size", len(spec))

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		slog.Debug("JSON unmarshal failed", "error", err)
		return err
	}
	slog.Debug("Unmarshaled JSON", "key_count", len(payload))

	vDst := reflect.ValueOf(dst).Elem()

	for path, fs := range spec {
		val, ok := traverse(payload, path)
		if !ok {
			slog.Debug("Path not found in payload", "path", path)
			continue // nothing supplied for this key
		}

		field := vDst.FieldByIndex(fs.index)
		if !field.CanSet() {
			slog.Debug("Field cannot be set", "path", path)
			continue
		}

		// Handle different field types
		switch field.Kind() {
		case reflect.String:
			if s, ok := val.(string); ok {
				field.SetString(s)
				slog.Debug("Set string field", "path", path, "value", s)
			}
		case reflect.Map, reflect.Struct, reflect.Interface:
			// For complex types, marshal back to JSON and unmarshal into the field
			b, err := json.Marshal(val)
			if err != nil {
				slog.Debug("Failed to marshal field value", "path", path, "error", err)
				continue
			}
			if err := json.Unmarshal(b, field.Addr().Interface()); err != nil {
				slog.Debug("Failed to unmarshal into field", "path", path, "error", err)
				return fmt.Errorf("%s: %w", path, err)
			}
			slog.Debug("Set complex field", "path", path)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if i, ok := val.(float64); ok { // JSON numbers are float64
				field.SetInt(int64(i))
				slog.Debug("Set int field", "path", path, "value", int64(i))
			}
		case reflect.Float32, reflect.Float64:
			if f, ok := val.(float64); ok {
				field.SetFloat(f)
				slog.Debug("Set float field", "path", path, "value", f)
			}
		case reflect.Bool:
			if b, ok := val.(bool); ok {
				field.SetBool(b)
				slog.Debug("Set bool field", "path", path, "value", b)
			}
		default:
			// Fallback: try JSON marshal/unmarshal
			b, err := json.Marshal(val)
			if err != nil {
				slog.Debug("Failed to marshal field value", "path", path, "error", err)
				continue
			}
			if err := json.Unmarshal(b, field.Addr().Interface()); err != nil {
				slog.Debug("Failed to unmarshal into field", "path", path, "error", err)
				return fmt.Errorf("%s: %w", path, err)
			}
			slog.Debug("Set field via fallback", "path", path)
		}
	}
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

// retryable executes a function with exponential backoff retry logic
func retryable(call func() error, max int, backoff time.Duration, log *slog.Logger) error {
	if max == 0 {
		return call() // no retry
	}

	delay := backoff
	for i := 0; i <= max; i++ {
		if err := call(); err != nil {
			if i == max {
				log.Debug("Final attempt failed", "attempt", i+1, "error", err)
				return err
			}
			log.Debug("Attempt failed, retrying", "attempt", i+1, "error", err, "delay", delay)
			time.Sleep(delay)
			delay *= 2
			continue
		}
		if i > 0 {
			log.Debug("Attempt succeeded", "attempt", i+1)
		}
		return nil
	}
	return nil
}
