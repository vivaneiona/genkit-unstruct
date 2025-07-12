package unstruct

import "strings"

// tagParts holds the two "segments" that may appear in `unstruct:"<prompt>,<model>"`.
type tagParts struct {
	prompt string // empty → inherit from parent
	model  string // empty → use Options.Model
}

// parseUnstructTag splits the tag string into prompt+model.
// Rules: 0-parts → inherit; 1-part → prompt *unless* it equals a known model id;
// 2-parts → prompt,model.
func parseUnstructTag(tag, inheritedPrompt string) (tp tagParts) {
	if tag == "" {
		tp.prompt = inheritedPrompt
		return
	}

	items := strings.Split(tag, ",")
	switch len(items) {
	case 1:
		if knownModel(items[0]) {
			tp.prompt, tp.model = inheritedPrompt, items[0] // model override only
		} else {
			tp.prompt = items[0] // prompt override
		}
	case 2:
		tp.prompt, tp.model = items[0], items[1]
	default:
		tp.prompt = inheritedPrompt // malformed → silently inherit
	}
	return
}

// knownModel checks if the given string is a known Gemini model variant
func knownModel(model string) bool {
	switch model {
	case "gemini-1.5-pro", "gemini-1.5-flash", "gemini-1.5-flash-8b",
		"gemini-2.5-pro", "gemini-1.0-pro", "gemini-pro", "gemini-flash":
		return true
	default:
		return false
	}
}
