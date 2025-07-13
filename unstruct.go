// Package unstruct implements a generic, multi-prompt extractor that can plug
// into any workflow engine via a tiny Runner abstraction.
package unstruct

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

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
	log.Debug("Raw response content", "content", part.Text)
	return []byte(part.Text), nil
}

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
	// No default fallback prompt - must be explicitly set
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

	// 1. Get new schema with grouping logic
	sch, err := schemaOf[T]()
	if err != nil {
		return nil, fmt.Errorf("schema analysis failed: %w", err)
	}
	x.log.Debug("Analyzed schema", "group_count", len(sch.group2keys), "field_count", len(sch.json2field))

	// 2. Fan-out prompt calls with improved grouping and model-specific handling.
	type frag struct {
		prompt string
		raw    []byte
		model  string
	}
	var (
		mu        sync.Mutex
		fragments = make([]frag, 0, len(sch.group2keys))
	)

	x.log.Debug("Starting concurrent prompt calls", "prompt_count", len(sch.group2keys))
	for pk, keys := range sch.group2keys {
		pk, keys := pk, keys // loop capture
		r.Go(func() error {
			model := opts.Model
			// Use model from promptKey if specified, otherwise check individual fields
			if pk.model != "" {
				model = pk.model
			} else if len(keys) == 1 {
				if m := sch.json2field[keys[0]].model; m != "" {
					model = m
				}
			}
			raw, err := x.callPrompt(egCtx, pk.prompt, keys, doc, model, opts)
			if err != nil {
				return fmt.Errorf("%s: %w", pk.prompt, err)
			}
			mu.Lock()
			fragments = append(fragments, frag{pk.prompt, raw, model})
			mu.Unlock()
			return nil
		})
	}

	if err := r.Wait(); err != nil {
		x.log.Debug("Prompt calls failed", "error", err)
		return nil, err
	}
	x.log.Debug("All prompt calls completed", "fragment_count", len(fragments))

	// 3. Merge JSON fragments back into a single struct using new patcher.
	var out T
	x.log.Debug("Starting JSON fragment merge", "fragment_count", len(fragments))
	for _, f := range fragments {
		x.log.Debug("Processing fragment", "prompt", f.prompt, "raw_content", string(f.raw))
		if err := patchStruct(&out, f.raw, sch.json2field); err != nil {
			x.log.Debug("Merge failed", "prompt", f.prompt, "error", err)
			return nil, fmt.Errorf("merge %q: %w", f.prompt, err)
		}
		x.log.Debug("Fragment merged successfully", "prompt", f.prompt)
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

// callPrompt invokes a single prompt template with a specific model and returns raw JSON bytes.
func (x *Unstructor[T]) callPrompt(
	ctx context.Context,
	promptLabel string,
	keys []string,
	doc string,
	model string,
	opts Options,
) ([]byte, error) {
	// label may be empty → check for fallback or error
	label := promptLabel
	if label == "" {
		if opts.FallbackPrompt == "" {
			return nil, fmt.Errorf("no prompt specified for fields %v and no fallback prompt provided - use WithFallbackPrompt() option", keys)
		}
		label = opts.FallbackPrompt
	}

	x.log.Debug("Calling prompt", "label", label, "keys", keys, "model", model)

	var tpl string
	var err error

	// Check if the prompt provider supports contextual prompts (like Stick templates)
	if contextProvider, ok := x.prompts.(ContextualPromptProvider); ok {
		tpl, err = contextProvider.GetPromptWithContext(label, 1, keys, doc)
	} else {
		tpl, err = x.prompts.GetPrompt(label, 1)
	}

	if err != nil {
		x.log.Debug("Failed to get prompt template", "label", label, "error", err)
		return nil, err
	}

	// For non-contextual providers, still use buildPrompt for {{.Keys}} replacement
	var prompt string
	if _, ok := x.prompts.(ContextualPromptProvider); ok {
		// Contextual providers already have variables replaced
		prompt = tpl
	} else {
		prompt = buildPrompt(tpl, keys, doc)
	}

	var result []byte
	err = retryable(func() error {
		var genErr error
		result, genErr = x.invoker.Generate(ctx, Model(model), prompt, opts.media)
		if genErr != nil {
			x.log.Debug("Generate failed", "label", label, "model", model, "error", genErr)
		}
		return genErr
	}, opts.MaxRetries, opts.Backoff, x.log)

	return result, err
}

// DryRun simulates the extraction process without making actual API calls.
// Returns execution statistics that can be used for planning and cost estimation.
func (x *Unstructor[T]) DryRun(
	ctx context.Context,
	doc string,
	optFns ...func(*Options),
) (*ExecutionStats, error) {
	if doc == "" {
		return nil, fmt.Errorf("dry run: %w", ErrEmptyDocument)
	}

	var opts Options
	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Model == "" {
		return nil, fmt.Errorf("dry run: %w", ErrModelMissing)
	}

	// Get schema analysis
	sch, err := schemaOf[T]()
	if err != nil {
		return nil, fmt.Errorf("schema analysis failed: %w", err)
	}

	// Collect statistics without making actual calls
	stats := &ExecutionStats{
		PromptCalls:       len(sch.group2keys),
		ModelCalls:        make(map[string]int),
		PromptGroups:      len(sch.group2keys),
		FieldsExtracted:   len(sch.json2field),
		GroupDetails:      make([]GroupExecution, 0, len(sch.group2keys)),
		TotalInputTokens:  0,
		TotalOutputTokens: 0,
	}

	x.log.Debug("Starting dry run analysis", "group_count", len(sch.group2keys), "field_count", len(sch.json2field))

	// Simulate the execution loop
	for pk, keys := range sch.group2keys {
		model := opts.Model
		// Use model from promptKey if specified, otherwise check individual fields
		if pk.model != "" {
			model = pk.model
		} else if len(keys) == 1 {
			if m := sch.json2field[keys[0]].model; m != "" {
				model = m
			}
		}

		// Get prompt template to estimate tokens
		tpl, err := x.prompts.GetPrompt(pk.prompt, 1)
		if err != nil {
			x.log.Debug("Failed to get prompt template", "prompt", pk.prompt, "error", err)
			// Use default template for estimation
			tpl = fmt.Sprintf("Extract the following fields from the document: %v", keys)
		}

		// Build the full prompt for token estimation
		fullPrompt := buildPrompt(tpl, keys, doc)

		// Estimate tokens
		inputTokens := EstimateTokensFromText(fullPrompt)
		outputTokens := estimateOutputTokensForFields(keys)

		// Update statistics
		stats.ModelCalls[model]++
		stats.TotalInputTokens += inputTokens
		stats.TotalOutputTokens += outputTokens

		// Add group details
		groupExec := GroupExecution{
			PromptName:   pk.prompt,
			Model:        model,
			Fields:       keys,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			ParentPath:   pk.parentPath,
		}
		stats.GroupDetails = append(stats.GroupDetails, groupExec)

		x.log.Debug("Simulated prompt call",
			"prompt", pk.prompt,
			"model", model,
			"fields", keys,
			"input_tokens", inputTokens,
			"output_tokens", outputTokens)
	}

	x.log.Info("Dry run completed",
		"prompt_calls", stats.PromptCalls,
		"total_input_tokens", stats.TotalInputTokens,
		"total_output_tokens", stats.TotalOutputTokens,
		"models_used", len(stats.ModelCalls))

	return stats, nil
}

// estimateOutputTokensForFields estimates output tokens based on field types and count
func estimateOutputTokensForFields(fields []string) int {
	// Base JSON structure overhead
	baseTokens := 10 + len(fields)*2 // {"field": "value", ...}

	// Estimate content tokens per field
	contentTokens := 0
	for _, field := range fields {
		switch {
		case containsField(field, "name") || containsField(field, "title"):
			contentTokens += 15 // Short text
		case containsField(field, "address") || containsField(field, "description"):
			contentTokens += 30 // Medium text
		case containsField(field, "email") || containsField(field, "phone") || containsField(field, "url"):
			contentTokens += 20 // Structured text
		case containsField(field, "age") || containsField(field, "count") || containsField(field, "number"):
			contentTokens += 5 // Numbers
		case containsField(field, "date") || containsField(field, "time"):
			contentTokens += 10 // Dates/times
		default:
			contentTokens += 20 // Default estimate
		}
	}

	return baseTokens + contentTokens
}

// containsField checks if a field name contains a substring (case-insensitive)
func containsField(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
