// Package unstruct implements a generic, multi-prompt extractor that can plug
// into any workflow engine via a tiny Runner abstraction.
package unstruct

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/genai"
)

// ErrEmptyDocument is returned when the source document is an empty string.
var ErrEmptyDocument = errors.New("document text is empty")
var ErrEmptyAssets = errors.New("no assets provided")
var ErrModelMissing = errors.New("model not specified")
var ErrMissingSchema = errors.New("schema is required")

// Message represents a message in a conversation
type Message struct {
	Role  string
	Parts []*Part
}

// NewUserMessage creates a new user message
func NewUserMessage(parts ...*Part) *Message {
	return &Message{Role: "user", Parts: parts}
}

// NewSystemMessage creates a new system message
func NewSystemMessage(parts ...*Part) *Message {
	return &Message{Role: "system", Parts: parts}
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

	// Build content from messages
	var contents []*genai.Content

	for _, msg := range cfg.Messages {
		var parts []*genai.Part

		for _, part := range msg.Parts {
			log.Debug("Processing message part", "type", part.Type, "file_uri", part.FileURI, "mime_type", part.MimeType)
			switch part.Type {
			case "text":
				// Add text part
				parts = append(parts, genai.NewPartFromText(part.Text))
			case "image":
				// Add image data part using Blob
				parts = append(parts, genai.NewPartFromBytes(part.Data, part.MimeType))
			case "file":
				// Add file part that references uploaded file using NewPartFromFile
				// This creates the proper file data part that the AI model can process
				file := genai.File{
					URI:      part.FileURI,
					MIMEType: part.MimeType,
				}
				genaiPart := genai.NewPartFromFile(file)
				log.Debug("Created genai file part", "uri", file.URI, "mime_type", file.MIMEType)
				parts = append(parts, genaiPart)
			}
		}

		if len(parts) > 0 {
			content := genai.NewContentFromParts(parts, genai.RoleUser)
			contents = append(contents, content)
		}
	}

	// Fallback to text-only if no messages provided
	if len(contents) == 0 {
		log.Debug("No valid content from messages")
		return nil, fmt.Errorf("no valid content provided")
	}

	log.Debug("Generating content", "model", modelName, "content_count", len(contents))

	// Create generation config for JSON output
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	}

	// Apply query parameters from tags if available
	if cfg.Parameters != nil {
		if temp, exists := cfg.Parameters["temperature"]; exists {
			tempFloat, err := strconv.ParseFloat(temp, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid temperature parameter '%s': %w", temp, err)
			}
			if tempFloat < 0 || tempFloat > 1 {
				return nil, fmt.Errorf("temperature parameter '%v' must be between 0.0 and 1.0", tempFloat)
			}
			val := float32(tempFloat)
			config.Temperature = &val
		}
		if topK, exists := cfg.Parameters["topK"]; exists {
			topKFloat, err := strconv.ParseFloat(topK, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid topK parameter '%s': %w", topK, err)
			}
			if topKFloat <= 0 {
				return nil, fmt.Errorf("topK parameter '%v' must be greater than 0", topKFloat)
			}
			val := float32(topKFloat)
			config.TopK = &val
		}
		if topP, exists := cfg.Parameters["topP"]; exists {
			topPFloat, err := strconv.ParseFloat(topP, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid topP parameter '%s': %w", topP, err)
			}
			if topPFloat < 0 || topPFloat > 1 {
				return nil, fmt.Errorf("topP parameter '%v' must be between 0.0 and 1.0", topPFloat)
			}
			val := float32(topPFloat)
			config.TopP = &val
		}
		if maxTokens, exists := cfg.Parameters["maxTokens"]; exists {
			maxTokensInt, err := strconv.Atoi(maxTokens)
			if err != nil {
				return nil, fmt.Errorf("invalid maxTokens parameter '%s': %w", maxTokens, err)
			}
			if maxTokensInt <= 0 {
				return nil, fmt.Errorf("maxTokens parameter '%d' must be greater than 0", maxTokensInt)
			}
			config.MaxOutputTokens = int32(maxTokensInt)
		}
		if maxOutputTokens, exists := cfg.Parameters["maxOutputTokens"]; exists {
			maxTokensInt, err := strconv.Atoi(maxOutputTokens)
			if err != nil {
				return nil, fmt.Errorf("invalid maxOutputTokens parameter '%s': %w", maxOutputTokens, err)
			}
			if maxTokensInt <= 0 {
				return nil, fmt.Errorf("maxOutputTokens parameter '%d' must be greater than 0", maxTokensInt)
			}
			config.MaxOutputTokens = int32(maxTokensInt)
		}
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
	assets []Asset,
	optFns ...func(*Options),
) (*T, error) {
	x.log.Debug("=== UNSTRUCT STARTED ===",
		"assets_count", len(assets),
		"options_count", len(optFns),
		"prompt_provider_type", fmt.Sprintf("%T", x.prompts))

	if len(assets) == 0 {
		x.log.Debug("No assets provided", "error", ErrEmptyAssets)
		return nil, fmt.Errorf("extract: %w", ErrEmptyAssets)
	}

	// Log assets details
	for i, asset := range assets {
		x.log.Debug("Asset details",
			"index", i,
			"asset_type", fmt.Sprintf("%T", asset))
	}

	var opts Options
	// No default fallback prompt - must be explicitly set
	for _, fn := range optFns {
		fn(&opts)
	}

	x.log.Debug("Options configured",
		"model", opts.Model,
		"timeout", opts.Timeout,
		"max_retries", opts.MaxRetries,
		"backoff", opts.Backoff)

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

	// 1. Get new schema with grouping logic and field model overrides
	sch, err := schemaOfWithOptions[T](&opts, x.log)
	if err != nil {
		return nil, fmt.Errorf("schema analysis failed: %w", err)
	}
	x.log.Debug("Analyzed schema", "group_count", len(sch.group2keys), "field_count", len(sch.json2field))

	// 2. Validate that at least one model is specified (either globally or per-field/group)
	hasModel := opts.Model != ""
	if !hasModel {
		// Check if any prompt group has a model specified
		for pk := range sch.group2keys {
			if pk.model != "" {
				hasModel = true
				break
			}
		}
		// If no group models, check individual fields
		if !hasModel {
			for _, fieldSpec := range sch.json2field {
				if fieldSpec.model != "" {
					hasModel = true
					break
				}
			}
		}
	}

	x.log.Debug("Starting extraction", "global_model", opts.Model, "has_model_somewhere", hasModel, "timeout", opts.Timeout, "max_retries", opts.MaxRetries)
	if !hasModel {
		return nil, fmt.Errorf("extract: %w", ErrModelMissing)
	}

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
		x.log.Debug("Processing prompt key", "prompt", pk.prompt, "model", pk.model, "keys", keys)
		r.Go(func() error {
			model := opts.Model
			var parameters map[string]string

			// Get parameters from group specs
			if groupSpec, exists := sch.group2specs[pk]; exists {
				parameters = groupSpec.parameters
				x.log.Debug("Got parameters from group spec", "parameters", parameters)
			} else {
				x.log.Debug("No group spec found for prompt key", "pk", pk)
			}

			// Use model from promptKey if specified, otherwise check individual fields
			if pk.model != "" {
				model = pk.model
			} else if len(keys) == 1 {
				if m := sch.json2field[keys[0]].model; m != "" {
					model = m
				}
				// Also get parameters from individual field if only one field
				if len(keys) == 1 && parameters == nil {
					parameters = sch.json2field[keys[0]].parameters
				}
			}
			raw, err := x.callPrompt(egCtx, pk.prompt, keys, assets, model, parameters, opts)
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

	// Convert string document to TextAsset
	assets := []Asset{&TextAsset{Content: doc}}

	result, err := d.Unstruct(ctx, assets, optFns...)
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

	// Convert string document to TextAsset
	assets := []Asset{&TextAsset{Content: doc}}

	// For now, streaming is a simplified implementation
	// In a full implementation, this would use the streaming API
	// and call onUpdate with partial results as they arrive
	result, err := x.Unstruct(ctx, assets, optFns...)
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
	assets []Asset,
	model string,
	parameters map[string]string,
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

	// Extract text content and media from assets for prompt building
	var textContent string
	var allMessages []*Message

	for _, asset := range assets {
		messages, err := asset.CreateMessages(ctx, x.log)
		if err != nil {
			return nil, fmt.Errorf("failed to create messages from asset: %w", err)
		}
		allMessages = append(allMessages, messages...)

		// Extract text content for template building (use first text part found)
		for _, msg := range messages {
			for _, part := range msg.Parts {
				if part.Type == "text" && textContent == "" {
					textContent = part.Text
				}
			}
		}
	}

	var tpl string
	var err error

	// Check if the prompt provider supports contextual prompts (like Stick templates)
	x.log.Debug("Checking prompt provider type",
		"provider_type", fmt.Sprintf("%T", x.prompts),
		"label", label,
		"keys", keys,
		"document_length", len(textContent),
		"document_preview", textContent[:min(100, len(textContent))])

	if contextProvider, ok := x.prompts.(ContextualPromptProvider); ok {
		x.log.Debug("Using ContextualPromptProvider (Stick/Twig)", "provider_type", fmt.Sprintf("%T", contextProvider))
		tpl, err = contextProvider.GetPromptWithContext(label, 1, keys, textContent)
		x.log.Debug("Got template from contextual provider",
			"template_length", len(tpl),
			"template_preview", tpl[:min(200, len(tpl))],
			"error", err)
	} else {
		x.log.Debug("Using basic PromptProvider", "provider_type", fmt.Sprintf("%T", x.prompts))
		tpl, err = x.prompts.GetPrompt(label, 1)
		x.log.Debug("Got template from basic provider",
			"template_length", len(tpl),
			"template_preview", tpl[:min(200, len(tpl))],
			"error", err)
	}

	if err != nil {
		x.log.Debug("Failed to get prompt template", "label", label, "error", err)
		return nil, err
	}

	// Use the template we already got
	prompt := tpl
	if _, ok := x.prompts.(ContextualPromptProvider); !ok {
		x.log.Debug("Using basic provider - replacing {{.Keys}} placeholder")
		// Replace keys placeholder manually for basic providers
		if strings.Contains(prompt, "{{.Keys}}") {
			keysStr := strings.Join(keys, ",")
			prompt = strings.ReplaceAll(prompt, "{{.Keys}}", keysStr)
			x.log.Debug("Replaced {{.Keys}} placeholder", "keys", keysStr)
		}
	}

	x.log.Debug("Final prompt constructed",
		"final_prompt_length", len(prompt),
		"final_prompt_preview", prompt[:min(300, len(prompt))])

	// Collect all media parts from all messages
	var mediaParts []*Part
	for _, msg := range allMessages {
		for _, part := range msg.Parts {
			if part.Type != "text" {
				mediaParts = append(mediaParts, part)
			}
		}
	}

	var result []byte
	err = retryable(func() error {
		var genErr error

		// If we have parameters, use GenerateBytes directly for better control
		if len(parameters) > 0 {
			// Build proper conversation messages
			var messages []*Message

			// Add system message with the template/instructions
			messages = append(messages, NewSystemMessage(NewTextPart(prompt)))

			// Add user messages with actual content
			for _, msg := range allMessages {
				userParts := append([]*Part(nil), msg.Parts...)
				if len(userParts) > 0 {
					messages = append(messages, NewUserMessage(userParts...))
				}
			}

			x.log.Debug("Built conversation",
				"message_count", len(messages),
				"system_prompt_length", len(prompt),
				"user_messages", len(allMessages))

			// Debug each message
			for i, msg := range messages {
				x.log.Debug("Message details",
					"index", i,
					"role", msg.Role,
					"parts_count", len(msg.Parts))
				for j, part := range msg.Parts {
					x.log.Debug("Part details",
						"message_index", i,
						"part_index", j,
						"type", part.Type,
						"text_preview", part.Text[:min(100, len(part.Text))])
				}
			}

			result, genErr = GenerateBytes(ctx, x.invoker.(*genkitInvoker).client, x.log,
				WithModelName(model),
				WithMessages(messages...),
				WithParameters(parameters),
			)
		} else {
			// Use the old path for backward compatibility
			result, genErr = x.invoker.Generate(ctx, Model(model), prompt, mediaParts)
		}

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
	assets []Asset,
	optFns ...func(*Options),
) (*ExecutionStats, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("dry run: %w", ErrEmptyAssets)
	}

	var opts Options
	for _, fn := range optFns {
		fn(&opts)
	}

	// Get schema analysis with field model overrides
	sch, err := schemaOfWithOptions[T](&opts, x.log)
	if err != nil {
		return nil, fmt.Errorf("schema analysis failed: %w", err)
	}

	// Validate that at least one model is specified (either globally or per-field/group)
	hasModel := opts.Model != ""
	if !hasModel {
		// Check if any prompt group has a model specified
		for pk := range sch.group2keys {
			if pk.model != "" {
				hasModel = true
				break
			}
		}
		// If no group models, check individual fields
		if !hasModel {
			for _, fieldSpec := range sch.json2field {
				if fieldSpec.model != "" {
					hasModel = true
					break
				}
			}
		}
	}

	if !hasModel {
		return nil, fmt.Errorf("dry run: %w", ErrModelMissing)
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

	// Extract text content from assets for token estimation
	var textContent string
	for _, asset := range assets {
		messages, err := asset.CreateMessages(ctx, x.log)
		if err != nil {
			continue // Skip assets that can't create messages
		}

		// Extract text content for template building (use first text part found)
		for _, msg := range messages {
			for _, part := range msg.Parts {
				if part.Type == "text" && textContent == "" {
					textContent = part.Text
					break
				}
			}
			if textContent != "" {
				break
			}
		}
		if textContent != "" {
			break
		}
	}

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
		fullPrompt := tpl
		// Replace keys placeholder manually for estimation
		if strings.Contains(fullPrompt, "{{.Keys}}") {
			keysStr := strings.Join(keys, ",")
			fullPrompt = strings.ReplaceAll(fullPrompt, "{{.Keys}}", keysStr)
		}

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

// Explain performs a dry run and returns a human-readable execution plan.
// This is a convenience method that combines DryRun with text formatting.
func (x *Unstructor[T]) Explain(
	ctx context.Context,
	assets []Asset,
	optFns ...func(*Options),
) (string, error) {
	// Get execution statistics from dry run
	stats, err := x.DryRun(ctx, assets, optFns...)
	if err != nil {
		return "", err
	}

	// Convert execution stats to a plan node
	plan := statsToPlan(stats)

	// Use the plan builder to format as text
	builder := NewPlanBuilder()
	return builder.formatAsText(plan), nil
}

// statsToPlan converts ExecutionStats to a PlanNode for formatting
func statsToPlan(stats *ExecutionStats) *PlanNode {
	// Create root SchemaAnalysis node
	fields := make([]string, 0)
	for _, group := range stats.GroupDetails {
		fields = append(fields, group.Fields...)
	}

	// Remove duplicates
	uniqueFields := extractUniqueStrings(fields)

	costConfig := DefaultCostCalculationConfig()
	rootNode := &PlanNode{
		Type:               SchemaAnalysisType,
		Fields:             uniqueFields,
		InputTokens:        10, // Schema analysis overhead
		EstCost:            float64(stats.PromptGroups) * costConfig.SchemaAnalysisBaseCost,
		Children:           make([]*PlanNode, 0),
		ExpectedModels:     make([]string, 0, len(stats.ModelCalls)),
		ExpectedCallCounts: stats.ModelCalls,
	}

	// Add models from stats
	for model := range stats.ModelCalls {
		rootNode.ExpectedModels = append(rootNode.ExpectedModels, model)
	}

	// Create PromptCall nodes from execution statistics
	for _, groupExec := range stats.GroupDetails {
		promptNode := &PlanNode{
			Type:         PromptCallType,
			PromptName:   groupExec.PromptName,
			Model:        groupExec.Model,
			Fields:       groupExec.Fields,
			InputTokens:  groupExec.InputTokens,
			OutputTokens: groupExec.OutputTokens,
			EstCost:      costConfig.PromptCallBaseCost + float64(groupExec.InputTokens)*costConfig.PromptCallTokenFactor,
			Children:     make([]*PlanNode, 0),
		}

		rootNode.Children = append(rootNode.Children, promptNode)
	}

	// Create MergeFragments node
	mergeNode := &PlanNode{
		Type:     MergeFragmentsType,
		Fields:   uniqueFields,
		EstCost:  costConfig.MergeFragmentsBaseCost + float64(len(uniqueFields))*costConfig.MergeFragmentsPerField,
		Children: make([]*PlanNode, 0),
	}
	rootNode.Children = append(rootNode.Children, mergeNode)

	// Update root cost to include children
	totalChildCost := 0.0
	for _, child := range rootNode.Children {
		totalChildCost += child.EstCost
	}
	rootNode.EstCost += totalChildCost

	return rootNode
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

// UnstructFromText is a convenience method for extracting from a single text document.
// This provides backward compatibility with the old string-based API.
func (x *Unstructor[T]) UnstructFromText(
	ctx context.Context,
	document string,
	optFns ...func(*Options),
) (*T, error) {
	assets := []Asset{NewTextAsset(document)}
	return x.Unstruct(ctx, assets, optFns...)
}

// DryRunFromText is a convenience method for dry-running a single text document.
// This provides backward compatibility with the old string-based API.
func (x *Unstructor[T]) DryRunFromText(
	ctx context.Context,
	document string,
	optFns ...func(*Options),
) (*ExecutionStats, error) {
	assets := []Asset{NewTextAsset(document)}
	return x.DryRun(ctx, assets, optFns...)
}

// ExplainFromText is a convenience method for explaining extraction from a single text document.
// This provides backward compatibility with the old string-based API.
func (x *Unstructor[T]) ExplainFromText(
	ctx context.Context,
	document string,
	optFns ...func(*Options),
) (string, error) {
	assets := []Asset{NewTextAsset(document)}
	return x.Explain(ctx, assets, optFns...)
}
