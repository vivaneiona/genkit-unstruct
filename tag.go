package unstruct

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var modelNameRegex = regexp.MustCompile(`(?i)^(gemini|gpt|claude)[\-\d\.]|^(vertex|googleai|openai)/|^(googleai|vertex)[\-\d\.]`)

type tagParts struct {
	prompt     string            // empty ⇒ inherit
	model      string            // empty ⇒ Options.Model
	parameters map[string]string // query parameters for model configuration
}

func parseUnstructTag(tag, inheritedPrompt string) (tp tagParts) {
	// inherit by default
	tp.prompt = inheritedPrompt
	tp.parameters = make(map[string]string)

	if tag == "" {
		return
	}

	fmt.Printf("DEBUG tag: Parsing tag '%s' with inherited prompt '%s'\n", tag, inheritedPrompt)

	// Fast path: no "/" and no "?" means simple tag
	if !strings.Contains(tag, "/") && !strings.Contains(tag, "?") {
		if strings.Contains(tag, ",") {
			if parts := strings.Split(tag, ","); len(parts) == 2 {
				tp.prompt, tp.model = parts[0], parts[1]
				return
			}
			// malformed comma format, fallback to inherited prompt
			return
		}
		// Single token - check if it looks like a model
		if looksLikeModel(tag) {
			tp.model = tag
		} else {
			tp.prompt = tag
		}
		return
	}

	// Parse as URL to handle both path and query parameters
	u, err := url.Parse(tag)
	if err != nil {
		// If URL parsing fails, treat as prompt
		tp.prompt = tag
		return
	}

	// Extract query parameters
	for key, values := range u.Query() {
		if len(values) > 0 {
			tp.parameters[key] = values[0] // take first value if multiple
		}
	}

	pathPart := u.Path

	// Handle legacy comma format in path
	if strings.Contains(pathPart, ",") {
		if parts := strings.Split(pathPart, ","); len(parts) == 2 {
			tp.prompt, tp.model = parts[0], parts[1]
			return
		}
		// malformed comma format, fallback to inherited prompt
		return
	}

	segs := strings.Split(pathPart, "/")

	// Process URL-style segments (prompt/name/model/name/etc)
	// Keys can only appear at the start or after another key/value pair
	for i := 0; i < len(segs); i++ {
		switch segs[i] {
		case "prompt":
			if i+1 < len(segs) {
				// Find the next key starting from appropriate position
				nextKeyIndex := len(segs)
				// Look for next key starting at the next even position after value
				startSearch := i + 2
				if startSearch%2 == 1 {
					startSearch++ // Ensure we start at even position
				}
				for j := startSearch; j < len(segs); j += 2 {
					if segs[j] == "prompt" || segs[j] == "model" || segs[j] == "group" {
						nextKeyIndex = j
						break
					}
				}
				// Join all segments between this key and the next key
				tp.prompt = strings.Join(segs[i+1:nextKeyIndex], "/")
			}
		case "model":
			if i+1 < len(segs) {
				// For simple case like model/value, take everything remaining
				if i == 0 {
					tp.model = strings.Join(segs[i+1:], "/")
				} else {
					// Find the next key starting from appropriate position
					nextKeyIndex := len(segs)
					// Look for next key starting at the next even position after value
					startSearch := i + 2
					if startSearch%2 == 1 {
						startSearch++ // Ensure we start at even position
					}
					for j := startSearch; j < len(segs); j += 2 {
						if segs[j] == "prompt" || segs[j] == "model" || segs[j] == "group" {
							nextKeyIndex = j
							break
						}
					}
					// Join all segments between this key and the next key
					tp.model = strings.Join(segs[i+1:nextKeyIndex], "/")
				}
			}
		case "group":
			if i+1 < len(segs) {
				// Find the next key starting from appropriate position
				nextKeyIndex := len(segs)
				// Look for next key starting at the next even position after value
				startSearch := i + 2
				if startSearch%2 == 1 {
					startSearch++ // Ensure we start at even position
				}
				for j := startSearch; j < len(segs); j += 2 {
					if segs[j] == "prompt" || segs[j] == "model" || segs[j] == "group" {
						nextKeyIndex = j
						break
					}
				}
				// Join all segments between this key and the next key
				tp.prompt = "group:" + strings.Join(segs[i+1:nextKeyIndex], "/")
			}
		default:
			// first segment without a recognised key →
			// decide if it *looks* like a model or a prompt override
			if i == 0 {
				if looksLikeModel(segs[0]) {
					tp.model = segs[0]
				} else {
					tp.prompt = segs[0]
				}
			}
		}
	}

	fmt.Printf("DEBUG tag: Result: prompt='%s', model='%s', parameters=%+v\n", tp.prompt, tp.model, tp.parameters)
	return
}

// looksLikeModel uses a regex to identify model strings more accurately.
func looksLikeModel(s string) bool {
	return modelNameRegex.MatchString(s)
}
