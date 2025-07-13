package unstruct

import "strings"

// tagParts holds the two "segments" that may appear in `unstruct:"<prompt>,<model>"`.
type tagParts struct {
	prompt string // empty → inherit from parent
	model  string // empty → use Options.Model
}

// parseUnstructTag splits the tag string into prompt+model.
// Supports multiple formats:
// - "" → inherit from parent
// - "PROMPTNAME,MODELNAME" → explicit prompt and model
// - "model/{modelname}" → model-only format
// - "prompt/{promptname}" → prompt-only format
// - "anything_else" → treated as prompt unless it looks like a model name
func parseUnstructTag(tag, inheritedPrompt string) (tp tagParts) {
	if tag == "" {
		tp.prompt = inheritedPrompt
		return
	}

	items := strings.Split(tag, ",")
	switch len(items) {
	case 1:
		// Check for special prefixes
		if strings.HasPrefix(items[0], "model/") {
			// model/{modelname} format - extract model, inherit prompt
			tp.prompt = inheritedPrompt
			tp.model = strings.TrimPrefix(items[0], "model/")
		} else if strings.HasPrefix(items[0], "prompt/") {
			// prompt/{promptname} format - extract prompt, no model specified
			tp.prompt = strings.TrimPrefix(items[0], "prompt/")
			tp.model = ""
		} else if looksLikeModel(items[0]) {
			// Bare model name - inherit prompt, set model
			tp.prompt = inheritedPrompt
			tp.model = items[0]
		} else {
			// Treat as prompt override (no model specified)
			tp.prompt = items[0]
		}
	case 2:
		// PROMPTNAME,MODELNAME format
		tp.prompt, tp.model = items[0], items[1]
	default:
		tp.prompt = inheritedPrompt // malformed → silently inherit
	}
	return
}

// looksLikeModel checks if a string looks like a model name
// This is a simple heuristic to detect common model naming patterns
func looksLikeModel(s string) bool {
	// Check for known Gemini models (bare names)
	if strings.Contains(s, "gemini") {
		return true
	}

	// Check for known provider prefixes with Gemini models
	if strings.HasPrefix(s, "googleai/") || strings.HasPrefix(s, "vertex/") {
		return true
	}

	return false
}
