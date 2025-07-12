package unstruct

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"
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

// patchStruct merges JSON data into the destination struct using dotted keys support
func patchStruct[T any](dst *T, raw []byte, spec map[string]fieldSpec) error {
	slog.Debug("Starting JSON merge", "raw_length", len(raw), "spec_size", len(spec))

	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		slog.Debug("JSON unmarshal failed", "error", err)
		return err
	}
	slog.Debug("Unmarshaled JSON", "key_count", len(m))

	for key, data := range m {
		fs, ok := spec[key]
		if !ok {
			slog.Debug("Field spec not found", "key", key)
			continue
		}
		v := reflect.ValueOf(dst).Elem()
		for _, idx := range fs.index {
			v = v.Field(idx)
		}
		if err := json.Unmarshal(data, v.Addr().Interface()); err != nil {
			slog.Debug("Field unmarshal failed", "key", key, "error", err)
			return fmt.Errorf("%s: %w", key, err)
		}
		slog.Debug("Patched field", "key", key)
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
