package unstruct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimplePromptProvider_GetPrompt(t *testing.T) {
	provider := SimplePromptProvider{
		"test":  "Test prompt for {{.Keys}}",
		"basic": "Basic prompt",
	}

	t.Run("existing prompt", func(t *testing.T) {
		prompt, err := provider.GetPrompt("test", 1)
		require.NoError(t, err)
		assert.Equal(t, "Test prompt for {{.Keys}}", prompt)
	})

	t.Run("non-existing prompt", func(t *testing.T) {
		prompt, err := provider.GetPrompt("nonexistent", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Empty(t, prompt)
	})
}

func TestWithTemplates(t *testing.T) {
	templates := map[string]string{
		"test":  "Test template",
		"basic": "Basic template",
	}

	provider, err := NewStickPromptProvider(WithTemplates(templates))
	require.NoError(t, err)

	prompt, err := provider.GetPrompt("test", 1)
	require.NoError(t, err)
	assert.Equal(t, "Test template", prompt)
}

func TestWithVar(t *testing.T) {
	templates := map[string]string{
		"test": "Test with {{customVar}}",
	}

	provider, err := NewStickPromptProvider(
		WithTemplates(templates),
		WithVar("customVar", "custom value"),
	)
	require.NoError(t, err)

	prompt, err := provider.GetPrompt("test", 1)
	require.NoError(t, err)
	assert.Equal(t, "Test with custom value", prompt)
}

func TestNewStickPromptProvider(t *testing.T) {
	t.Run("empty provider", func(t *testing.T) {
		provider, err := NewStickPromptProvider()
		require.NoError(t, err)
		assert.NotNil(t, provider)

		// Should return error for non-existent template
		_, err = provider.GetPrompt("nonexistent", 1)
		assert.Error(t, err)
	})

	t.Run("with templates", func(t *testing.T) {
		templates := map[string]string{
			"test": "Hello {{tag}}",
		}

		provider, err := NewStickPromptProvider(WithTemplates(templates))
		require.NoError(t, err)

		prompt, err := provider.GetPrompt("test", 1)
		require.NoError(t, err)
		assert.Equal(t, "Hello test", prompt)
	})
}

func TestStickPromptProvider_AddTemplate(t *testing.T) {
	provider, err := NewStickPromptProvider()
	require.NoError(t, err)

	provider.AddTemplate("new", "New template")

	prompt, err := provider.GetPrompt("new", 1)
	require.NoError(t, err)
	assert.Equal(t, "New template", prompt)
}

func TestStickPromptProvider_GetPrompt(t *testing.T) {
	templates := map[string]string{
		"basic":   "Basic template for {{Tag}} version {{version}}",
		"complex": "Complex template with {{Tag}} and {{Version}}",
	}

	provider, err := NewStickPromptProvider(WithTemplates(templates))
	require.NoError(t, err)

	t.Run("basic template", func(t *testing.T) {
		prompt, err := provider.GetPrompt("basic", 2)
		require.NoError(t, err)
		assert.Equal(t, "Basic template for basic version 2", prompt)
	})

	t.Run("complex template", func(t *testing.T) {
		prompt, err := provider.GetPrompt("complex", 3)
		require.NoError(t, err)
		assert.Equal(t, "Complex template with complex and 3", prompt)
	})

	t.Run("non-existent template", func(t *testing.T) {
		_, err := provider.GetPrompt("nonexistent", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestStickPromptProvider_GetPromptWithContext(t *testing.T) {
	templates := map[string]string{
		"extraction": "Extract {{KeyList}} from:\n{{Document}}",
		"analysis":   "Analyze {{KeyList}} in document:\n{{document}}",
	}

	provider, err := NewStickPromptProvider(WithTemplates(templates))
	require.NoError(t, err)

	t.Run("extraction template", func(t *testing.T) {
		keys := []string{"name", "email", "phone"}
		document := "John Doe, john@email.com, 555-1234"

		prompt, err := provider.GetPromptWithContext("extraction", 1, keys, document)
		require.NoError(t, err)

		expected := "Extract name, email, phone from:\nJohn Doe, john@email.com, 555-1234"
		assert.Equal(t, expected, prompt)
	})

	t.Run("analysis template", func(t *testing.T) {
		keys := []string{"age", "location"}
		document := "Person is 30 years old from New York"

		prompt, err := provider.GetPromptWithContext("analysis", 1, keys, document)
		require.NoError(t, err)

		expected := "Analyze age, location in document:\nPerson is 30 years old from New York"
		assert.Equal(t, expected, prompt)
	})

	t.Run("non-existent template", func(t *testing.T) {
		_, err := provider.GetPromptWithContext("nonexistent", 1, []string{}, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
