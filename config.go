package unstruct

// GenerateOption represents options for generation
type GenerateOption func(*generateConfig)

type generateConfig struct {
	ModelName string
	Messages  []*Message
}

// WithModelName sets the model name
func WithModelName(name string) GenerateOption {
	return func(cfg *generateConfig) {
		cfg.ModelName = name
	}
}

// WithMessages sets the messages
func WithMessages(messages ...*Message) GenerateOption {
	return func(cfg *generateConfig) {
		cfg.Messages = messages
	}
}
