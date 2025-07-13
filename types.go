package unstruct

import (
	"context"
	"reflect"
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

// ContextualPromptProvider extends PromptProvider to support template variables.
type ContextualPromptProvider interface {
	PromptProvider
	GetPromptWithContext(tag string, version int, keys []string, document string) (string, error)
}

// Invoker abstraction allows mocking, retrying, and caching
type Invoker interface {
	Generate(ctx context.Context, model Model, prompt string, media []*Part) ([]byte, error)
}

// FieldModelMap represents model overrides for specific type and field combinations
type FieldModelMap map[string]string // key: "TypeName.FieldName", value: model name

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
	FieldModels      FieldModelMap             // per-field model overrides
	FlattenGroups    bool                      // if true, ignore parent paths when grouping by prompt+model
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

func WithFallbackPrompt(prompt string) func(*Options) {
	return func(o *Options) { o.FallbackPrompt = prompt }
}

// WithModelFor sets a specific model for a particular field of a given type
// Usage: WithModelFor("gemini-1.5-pro", SomeType{}, "FieldName")
func WithModelFor(model string, typ any, fieldName string) func(*Options) {
	return func(o *Options) {
		if o.FieldModels == nil {
			o.FieldModels = make(FieldModelMap)
		}
		typeName := reflect.TypeOf(typ).Name()
		key := typeName + "." + fieldName
		o.FieldModels[key] = model
	}
}

// WithFlattenGroups enables flattening of groups with the same prompt and model
// When enabled, fields with the same prompt and model will be grouped together
// regardless of their parent path, resulting in fewer API calls
func WithFlattenGroups() func(*Options) {
	return func(o *Options) { o.FlattenGroups = true }
}
