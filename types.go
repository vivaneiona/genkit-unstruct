package unstruct

import (
	"context"
	"time"
)

// Model represents a model identifier
type Model string

// Runner lets Unstructor schedule work with any concurrency model.
type Runner interface {
	Go(fn func() error) // schedule
	Wait() error        // join / propagate first err
}

// PromptProvider should return the prompt template text for the given tag
type PromptProvider interface {
	GetPrompt(tag string, version int) (string, error)
}

// Invoker abstraction allows mocking, retrying, and caching
type Invoker interface {
	Generate(ctx context.Context, model Model, prompt string, media []*Part) ([]byte, error)
}

// Options represents functional options for extraction
type Options struct {
	Model            string
	Timeout          time.Duration
	Runner           Runner                    // nil → DefaultRunner
	OutputSchemaJSON string                    // optional JSON-Schema
	Streaming        bool                      // opt-in
	MaxRetries       int                       // 0 → no retry
	Backoff          time.Duration             // backoff duration for retries
	CustomParser     func([]byte) (any, error) // override JSON→struct
	FallbackPrompt   string                    // used when tag.prompt == ""
	media            []*Part                   // internal use
}

// Functional option constructors
func WithModel(name string) func(*Options) {
	return func(o *Options) { o.Model = name }
}

func WithTimeout(d time.Duration) func(*Options) {
	return func(o *Options) { o.Timeout = d }
}

func WithRunner(r Runner) func(*Options) {
	return func(o *Options) { o.Runner = r }
}

func WithOutputSchema(schema string) func(*Options) {
	return func(o *Options) { o.OutputSchemaJSON = schema }
}

func WithStreaming() func(*Options) {
	return func(o *Options) { o.Streaming = true }
}

func WithRetry(max int, backoff time.Duration) func(*Options) {
	return func(o *Options) {
		o.MaxRetries = max
		o.Backoff = backoff
	}
}

func WithParser(fn func([]byte) (any, error)) func(*Options) {
	return func(o *Options) { o.CustomParser = fn }
}

func WithMedia(p ...*Part) func(*Options) {
	return func(o *Options) { o.media = p }
}

func WithFallbackPrompt(prompt string) func(*Options) {
	return func(o *Options) { o.FallbackPrompt = prompt }
}
