package temporal_demo

import (
	"context"

	"google.golang.org/genai"
)

func CreateDefaultGenAIClient(ctx context.Context, apiKey string) (*genai.Client, error) {
	return genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
}
