package unstruct

import (
	"context"
	"encoding/json"
	"log/slog"
)

// testInvoker is a mock invoker for testing
type testInvoker struct{}

func (t *testInvoker) Generate(
	ctx context.Context,
	model Model,
	prompt string,
	media []*Part,
) ([]byte, error) {
	// Return mock JSON response for testing
	mockResponse := map[string]interface{}{
		"name": "Test Project",
		"code": "TEST-123",
		"lat":  40.7128,
		"lon":  -74.0060,
	}
	return json.Marshal(mockResponse)
}

// NewForTesting creates a Unstructor with a test invoker that doesn't require a real client
func NewForTesting[T any](p PromptProvider) *Unstructor[T] {
	return &Unstructor[T]{
		invoker: &testInvoker{},
		prompts: p,
		log:     slog.Default(),
	}
}
