package unstruct

import (
	"net/url"
	"strings"
)

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

	// Check for legacy comma format first (no URL parsing needed)
	if !strings.Contains(tag, "/") && !strings.Contains(tag, "?") {
		if strings.Contains(tag, ",") {
			if parts := strings.Split(tag, ","); len(parts) == 2 {
				tp.prompt, tp.model = parts[0], parts[1]
				return
			} else if len(parts) > 2 {
				// malformed comma format, fallback to inherited prompt
				return
			}
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
		} else if len(parts) > 2 {
			// malformed comma format, fallback to inherited prompt
			return
		}
	}

	// path-style (slash-separated) or single token
	segs := strings.Split(pathPart, "/")

	// Special handling for simple prefix cases (backward compatibility)
	if len(segs) >= 2 {
		firstSeg := segs[0]
		if firstSeg == "model" || firstSeg == "prompt" || firstSeg == "group" {
			// Check if this looks like a simple prefix case (not URL-style)
			// URL-style should have multiple different keys
			hasMultipleKeys := false
			seenKeys := make(map[string]bool)

			for i := 0; i < len(segs); i += 2 {
				if i < len(segs) && (segs[i] == "prompt" || segs[i] == "model" || segs[i] == "group") {
					seenKeys[segs[i]] = true
				}
			}

			// Only treat as URL-style if we have multiple different key types
			if len(seenKeys) > 1 {
				hasMultipleKeys = true
			}

			if !hasMultipleKeys {
				// Simple prefix case - everything after first segment is the value
				value := strings.Join(segs[1:], "/")
				switch firstSeg {
				case "model":
					tp.model = value
				case "prompt":
					tp.prompt = value
				case "group":
					tp.prompt = "group:" + value
				}
				return
			}
		}
	}

	// iterate through slash segments looking for keys (URL-style format)
	for i := 0; i < len(segs); i++ {
		switch segs[i] {
		case "prompt":
			if i+1 < len(segs) {
				// Find the next key or end of segments
				nextKeyIndex := len(segs)
				for j := i + 2; j < len(segs); j++ {
					if segs[j] == "prompt" || segs[j] == "model" || segs[j] == "group" {
						nextKeyIndex = j
						break
					}
				}
				// Join all segments between this key and the next key
				tp.prompt = strings.Join(segs[i+1:nextKeyIndex], "/")
				i = nextKeyIndex - 1 // -1 because loop will increment
			}
		case "model":
			if i+1 < len(segs) {
				// Find the next key or end of segments
				nextKeyIndex := len(segs)
				for j := i + 2; j < len(segs); j++ {
					if segs[j] == "prompt" || segs[j] == "model" || segs[j] == "group" {
						nextKeyIndex = j
						break
					}
				}
				// Join all segments between this key and the next key
				tp.model = strings.Join(segs[i+1:nextKeyIndex], "/")
				i = nextKeyIndex - 1 // -1 because loop will increment
			}
		case "group":
			if i+1 < len(segs) {
				// Find the next key or end of segments
				nextKeyIndex := len(segs)
				for j := i + 2; j < len(segs); j++ {
					if segs[j] == "prompt" || segs[j] == "model" || segs[j] == "group" {
						nextKeyIndex = j
						break
					}
				}
				// Join all segments between this key and the next key
				tp.prompt = "group:" + strings.Join(segs[i+1:nextKeyIndex], "/")
				i = nextKeyIndex - 1 // -1 because loop will increment
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

	return
}

// simple heuristic for model strings
func looksLikeModel(s string) bool {
	if strings.Contains(s, "gemini") ||
		strings.HasPrefix(s, "vertex:") ||
		strings.HasPrefix(s, "googleai/") ||
		strings.HasPrefix(s, "vertex/") ||
		strings.Contains(s, ":gemini") {
		return true
	}
	return false
}
