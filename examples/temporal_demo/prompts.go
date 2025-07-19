package temporal_demo

import (
	"embed"
	"log"

	unstruct "github.com/vivaneiona/genkit-unstruct"
)

//go:embed templates/*.twig
var templateFS embed.FS

// Prompts provides Twig template mappings for the temporal demo.
// Templates are loaded from the embedded templates directory.
var Prompts = createPromptProvider()

func createPromptProvider() unstruct.PromptProvider {
	provider, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(templateFS, "templates"),
	)
	if err != nil {
		log.Fatalf("Failed to create prompt provider: %v", err)
	}
	return provider
}
