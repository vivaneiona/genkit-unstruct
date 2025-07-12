// Package unstruct implements a generic, multi-prompt extractor that can plug
// into any workflow engine via a tiny Runner abstraction.
package unstruct

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/genai"
)

// ErrEmptyDocument is returned when the source document is an empty string.
var ErrEmptyDocument = errors.New("document text is empty")
var ErrModelMissing = errors.New("model not specified")
var ErrMissingSchema = errors.New("schema is required")

// Part represents a part of a message (text, image, etc.)
type Part struct {
	Type string
	Text string
	Data []byte
}

// NewTextPart creates a new text part
func NewTextPart(text string) *Part {
	return &Part{Type: "text", Text: text}
}

// Message represents a message in a conversation
type Message struct {
	Role  string
	Parts []*Part
}

// NewUserMessage creates a new user message
func NewUserMessage(parts ...*Part) *Message {
	return &Message{Role: "user", Parts: parts}
}

// Unstructor provides multi-prompt extraction capabilities.
type Unstructor[T any] struct {
	invoker Invoker
	prompts PromptProvider
	log     *slog.Logger
}

// GenerateBytes generates bytes using the Gemini API via Google GenAI
func GenerateBytes(ctx context.Context, client *genai.Client, log *slog.Logger, opts ...GenerateOption) ([]byte, error) {
	var cfg generateConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	if client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	// Get the model from config or use default
	modelName := cfg.ModelName
	if modelName == "" {
		modelName = "gemini-1.5-pro"
	}

	// Build the prompt from messages
	var prompt string
	if len(cfg.Messages) > 0 && len(cfg.Messages[0].Parts) > 0 {
		prompt = cfg.Messages[0].Parts[0].Text
	}

	if prompt == "" {
		log.Debug("No prompt provided")
		return nil, fmt.Errorf("no prompt provided")
	}

	log.Debug("Generating content", "model", modelName, "prompt_length", len(prompt))

	// Create content for the request
	contents := []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}

	// Create generation config for JSON output
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	}

	// Generate content
	resp, err := client.Models.GenerateContent(ctx, modelName, contents, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	log.Debug("Received response", "candidates_count", len(resp.Candidates))

	// Extract the response text
	if len(resp.Candidates) == 0 {
		log.Debug("No candidates in response")
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no parts in candidate content")
	}

	// Get the text from the first part
	part := candidate.Content.Parts[0]
	if part.Text == "" {
		return nil, fmt.Errorf("no text in first part of response")
	}

	log.Debug("Generated content successfully", "response_length", len(part.Text))
	return []byte(part.Text), nil
}

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

// DefaultRunner returns the default implementation backed by errgroup.Group.
func DefaultRunner(ctx context.Context) Runner {
	return newErrGroupRunner(ctx)
}

// errGroupRunner is the default implementation backed by errgroup.Group.
type errGroupRunner struct {
	ctx context.Context // derived ctx shared by all tasks
	eg  *errgroup.Group
}

func newErrGroupRunner(parent context.Context) *errGroupRunner {
	eg, ctx := errgroup.WithContext(parent)
	return &errGroupRunner{ctx: ctx, eg: eg}
}

func (r *errGroupRunner) Go(fn func() error) { r.eg.Go(fn) }
func (r *errGroupRunner) Wait() error        { return r.eg.Wait() }

// New returns a Unstructor that logs with slog.Default().
func New[T any](client *genai.Client, p PromptProvider) *Unstructor[T] {
	return NewWithLogger[T](client, p, slog.Default())
}

// NewWithLogger lets the caller supply their own logger.
func NewWithLogger[T any](client *genai.Client, p PromptProvider, log *slog.Logger) *Unstructor[T] {
	if log == nil {
		log = slog.Default()
	}
	return &Unstructor[T]{invoker: &genkitInvoker{client: client, log: log}, prompts: p, log: log}
}

// Unstruct runs the multi-prompt flow and returns a fully-populated value.
func (x *Unstructor[T]) Unstruct(
	ctx context.Context,
	doc string,
	optFns ...func(*Options),
) (*T, error) {
	if doc == "" {
		return nil, fmt.Errorf("extract: %w", ErrEmptyDocument)
	}

	var opts Options
	for _, fn := range optFns {
		fn(&opts)
	}

	x.log.Debug("Starting extraction", "model", opts.Model, "timeout", opts.Timeout, "max_retries", opts.MaxRetries)
	if opts.Model == "" {
		return nil, fmt.Errorf("extract: %w", ErrModelMissing)
	}

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
		x.log.Debug("Set timeout", "timeout", opts.Timeout)
	}

	// Decide which runner to use.
	r := opts.Runner
	if r == nil {
		r = DefaultRunner(ctx)
		x.log.Debug("Using default runner")
	}

	// Use the derived ctx if we're on the default runner; otherwise fall back.
	egCtx := ctx
	if d, ok := r.(*errGroupRunner); ok {
		egCtx = d.ctx
	}

	// 1. Cached schema → tag→keys, json→field map.
	tag2keys, json2field := schemaOf[T]()
	x.log.Debug("Analyzed schema", "tag_count", len(tag2keys), "field_count", len(json2field))

	// 2. Fan-out prompt calls.
	type frag struct {
		tag string
		raw []byte
	}
	var (
		mu        sync.Mutex
		fragments = make([]frag, 0, len(tag2keys))
	)

	x.log.Debug("Starting concurrent prompt calls", "prompt_count", len(tag2keys))
	for tag, keys := range tag2keys {
		tag, keys := tag, keys // loop capture
		r.Go(func() error {
			raw, err := x.callPrompt(egCtx, tag, keys, doc, opts)
			if err != nil {
				x.log.Debug("Prompt call failed", "tag", tag, "error", err)
				return fmt.Errorf("%s: %w", tag, err)
			}
			mu.Lock()
			fragments = append(fragments, frag{tag, raw})
			mu.Unlock()
			return nil
		})
	}

	if err := r.Wait(); err != nil {
		x.log.Debug("Prompt calls failed", "error", err)
		return nil, err
	}
	x.log.Debug("All prompt calls completed", "fragment_count", len(fragments))

	// 3. Merge JSON fragments back into a single struct.
	var out T
	x.log.Debug("Starting JSON fragment merge", "fragment_count", len(fragments))
	for _, f := range fragments {
		if err := patchStruct(&out, f.raw, json2field); err != nil {
			x.log.Debug("Merge failed", "tag", f.tag, "error", err)
			return nil, fmt.Errorf("merge %q: %w", f.tag, err)
		}
	}

	x.log.Info("Extraction completed successfully", "type", fmt.Sprintf("%T", out))
	return &out, nil
}

// DynamicUnstructor is a specialized unstructor for dynamic schema extraction
type DynamicUnstructor struct {
	*Unstructor[map[string]any]
}

// NewDynamic creates a DynamicUnstructor for schema-driven extraction
func NewDynamic(client *genai.Client, p PromptProvider, log *slog.Logger) *DynamicUnstructor {
	if log == nil {
		log = slog.Default()
	}
	log.Debug("Creating DynamicUnstructor")
	return &DynamicUnstructor{
		Unstructor: NewWithLogger[map[string]any](client, p, log),
	}
}

// ExtractDynamic performs dynamic schema extraction using JSON-Schema string.
// Returns a generic map result instead of a strongly-typed struct.
func (d *DynamicUnstructor) ExtractDynamic(
	ctx context.Context,
	doc string,
	schema string,
	optFns ...func(*Options),
) (map[string]any, error) {
	d.log.Debug("Starting dynamic extraction", "document_length", len(doc), "schema_length", len(schema), "options_count", len(optFns))

	if doc == "" {
		return nil, fmt.Errorf("extract dynamic: %w", ErrEmptyDocument)
	}

	if schema == "" {
		return nil, fmt.Errorf("extract dynamic: %w", ErrMissingSchema)
	}

	// Add schema to options
	optFns = append(optFns, WithOutputSchema(schema))

	result, err := d.Unstructor.Unstruct(ctx, doc, optFns...)
	if err != nil {
		return nil, err
	}

	d.log.Debug("Extraction completed successfully", "result_keys", len(*result))
	return *result, nil
}

// Stream performs streaming extraction with partial updates.
// The onUpdate callback receives successive partial structs and should return
// false to stop early, true to continue.
func (x *Unstructor[T]) Stream(
	ctx context.Context,
	doc string,
	onUpdate func(partial *T) bool,
	optFns ...func(*Options),
) error {
	x.log.Debug("Starting streaming extraction", "document_length", len(doc))
	if doc == "" {
		return fmt.Errorf("stream: %w", ErrEmptyDocument)
	}

	// Enable streaming in options
	optFns = append(optFns, WithStreaming())

	// For now, streaming is a simplified implementation
	// In a full implementation, this would use the streaming API
	// and call onUpdate with partial results as they arrive
	result, err := x.Unstruct(ctx, doc, optFns...)
	if err != nil {
		return err
	}

	// Call onUpdate with final result
	onUpdate(result)
	return nil
}

// retryable executes a function with exponential backoff retry logic
func retryable(call func() error, max int, backoff time.Duration, log *slog.Logger) error {
	if max == 0 {
		return call() // no retry
	}

	delay := backoff
	for i := 0; i <= max; i++ {
		if err := call(); err != nil {
			if i == max {
				log.Debug("Final attempt failed", "attempt", i+1, "error", err)
				return err
			}
			log.Debug("Attempt failed, retrying", "attempt", i+1, "error", err, "delay", delay)
			time.Sleep(delay)
			delay *= 2
			continue
		}
		if i > 0 {
			log.Debug("Attempt succeeded", "attempt", i+1)
		}
		return nil
	}
	return nil
}

// genkitInvoker implements the Invoker interface using Google GenAI
type genkitInvoker struct {
	client *genai.Client
	log    *slog.Logger
}

func (gv *genkitInvoker) Generate(
	ctx context.Context,
	model Model,
	prompt string,
	media []*Part,
) ([]byte, error) {
	gv.log.Debug("Starting generation", "model", string(model), "prompt_length", len(prompt), "media_count", len(media))

	if gv.client == nil {
		gv.log.Debug("Client not initialized")
		return nil, fmt.Errorf("client not initialized")
	}

	return GenerateBytes(ctx, gv.client, gv.log,
		WithModelName(string(model)),
		WithMessages(NewUserMessage(
			append([]*Part{NewTextPart(prompt)}, media...)...,
		)),
	)
}

// callPrompt invokes a single prompt template and returns raw JSON bytes.
func (x *Unstructor[T]) callPrompt(
	ctx context.Context,
	tag string,
	keys []string,
	doc string,
	opts Options,
) ([]byte, error) {
	x.log.Debug("Calling prompt", "tag", tag, "keys", keys)

	tpl, err := x.prompts.GetPrompt(tag, 1)
	if err != nil {
		x.log.Debug("Failed to get prompt template", "tag", tag, "error", err)
		return nil, err
	}

	prompt := buildPrompt(tpl, keys, doc)

	var result []byte
	err = retryable(func() error {
		var genErr error
		result, genErr = x.invoker.Generate(ctx, Model(opts.Model), prompt, opts.media)
		if genErr != nil {
			x.log.Debug("Generate failed", "tag", tag, "error", genErr)
		}
		return genErr
	}, opts.MaxRetries, opts.Backoff, x.log)

	return result, err
}
