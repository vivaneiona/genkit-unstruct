package unstruct

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"
)

func TestParameterValidation(t *testing.T) {
	// Test invalid temperature parameter
	t.Run("invalid temperature string", func(t *testing.T) {
		_, err := GenerateBytes(context.Background(), &genai.Client{}, slog.Default(),
			WithModelName("gemini-1.5-pro"),
			WithMessages(NewUserMessage(NewTextPart("test"))),
			WithParameters(map[string]string{"temperature": "invalid"}),
		)
		assert.Error(t, err, "Expected error for invalid temperature parameter")
		assert.Contains(t, err.Error(), "invalid temperature parameter")
	})

	t.Run("temperature out of range", func(t *testing.T) {
		_, err := GenerateBytes(context.Background(), &genai.Client{}, slog.Default(),
			WithModelName("gemini-1.5-pro"),
			WithMessages(NewUserMessage(NewTextPart("test"))),
			WithParameters(map[string]string{"temperature": "1.5"}),
		)
		assert.Error(t, err, "Expected error for temperature out of range")
		errMsg := err.Error()
		containsValidation := assert.Contains(t, errMsg, "temperature", "Error should mention temperature") ||
			assert.Contains(t, errMsg, "must be between", "Error should mention valid range")
		assert.True(t, containsValidation, "Expected temperature validation error, got: %v", err)
	})

	t.Run("invalid topK parameter", func(t *testing.T) {
		_, err := GenerateBytes(context.Background(), &genai.Client{}, slog.Default(),
			WithModelName("gemini-1.5-pro"),
			WithMessages(NewUserMessage(NewTextPart("test"))),
			WithParameters(map[string]string{"topK": "invalid"}),
		)
		assert.Error(t, err, "Expected error for invalid topK parameter")
		assert.Contains(t, err.Error(), "invalid topK parameter")
	})

	t.Run("topK zero or negative", func(t *testing.T) {
		_, err := GenerateBytes(context.Background(), &genai.Client{}, slog.Default(),
			WithModelName("gemini-1.5-pro"),
			WithMessages(NewUserMessage(NewTextPart("test"))),
			WithParameters(map[string]string{"topK": "0"}),
		)
		assert.Error(t, err, "Expected error for topK <= 0")
		errMsg := err.Error()
		containsValidation := assert.Contains(t, errMsg, "topK", "Error should mention topK") ||
			assert.Contains(t, errMsg, "must be greater than 0", "Error should mention valid range")
		assert.True(t, containsValidation, "Expected topK validation error, got: %v", err)
	})

	t.Run("invalid maxTokens parameter", func(t *testing.T) {
		_, err := GenerateBytes(context.Background(), &genai.Client{}, slog.Default(),
			WithModelName("gemini-1.5-pro"),
			WithMessages(NewUserMessage(NewTextPart("test"))),
			WithParameters(map[string]string{"maxTokens": "invalid"}),
		)
		assert.Error(t, err, "Expected error for invalid maxTokens parameter")
		assert.Contains(t, err.Error(), "invalid maxTokens parameter")
	})
}
