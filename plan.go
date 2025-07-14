// Package unstruct provides execution planning for unstructured data extraction.
//
// The plan module helps analyze and estimate costs for extraction operations before
// executing them. It provides detailed execution plans with token estimates and
// cost calculations for different LLM models.
//
// # Basic Usage
//
// Create a plan for a simple schema:
//
//	schema := map[string]interface{}{
//		"fields": []string{"name", "email", "company"},
//	}
//
//	plan, err := NewPlanBuilder().
//		WithSchema(schema).
//		Explain()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Format as text
//	textPlan, _ := NewPlanBuilder().
//		WithSchema(schema).
//		ExplainPretty(FormatText)
//	fmt.Println(textPlan)
//
// # Cost Estimation
//
// Get real cost estimates with model pricing:
//
//	pricing := DefaultModelPricing()
//
//	costPlan, err := NewPlanBuilder().
//		WithSchema(schema).
//		ExplainPrettyWithCosts(FormatText, pricing)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(costPlan)
//
// # Advanced Configuration
//
// Configure models and prompts for specific fields:
//
//	modelConfig := map[string]string{
//		"email": "gpt-4o-mini",
//		"name":  "gemini-1.5-flash",
//	}
//
//	promptConfig := map[string]interface{}{
//		"email": "EmailExtractionPrompt",
//		"name":  "NameExtractionPrompt",
//	}
//
//	plan, err := NewPlanBuilder().
//		WithSchema(schema).
//		WithModelConfig(modelConfig).
//		WithPromptConfig(promptConfig).
//		ExplainWithCosts(pricing)
//
// # Dry-Run Execution
//
// For more accurate plans, use dry-run execution with an actual Unstructor:
//
//	unstructor := NewUnstructor(schema, WithModel("gpt-4o"))
//	sampleDoc := "John Doe\njohn@example.com\nAcme Corp"
//
//	plan, err := NewPlanBuilder().
//		WithSchema(schema).
//		WithUnstructor(unstructor).
//		WithSampleDocument(sampleDoc).
//		Explain()
//
// # Output Formats
//
// Plans can be formatted as text trees or JSON:
//
//	// Text format (ASCII tree)
//	textOutput, _ := builder.ExplainPretty(FormatText)
//
//	// JSON format
//	jsonOutput, _ := builder.ExplainPretty(FormatJSON)
package unstruct

import (
	"context"
	"fmt"
	"strings"
)

// Token estimation constants
const (
	CharsPerToken      = 4   // Average characters per token for English text
	TokensPerWordRatio = 1.3 // Average tokens per word
	BasePromptTokens   = 50  // Base tokens for prompt template
	DocumentTokens     = 200 // Tokens for document context
	SchemaBaseTokens   = 20  // Base overhead for schema analysis
	TokensPerField     = 5   // Additional tokens per field in schema
)

// Cost calculation constants
const (
	SchemaAnalysisBaseCost = 1.0  // Base cost for schema analysis
	SchemaAnalysisPerField = 0.5  // Additional cost per field
	PromptCallBaseCost     = 3.0  // Base cost for prompt calls
	PromptCallTokenFactor  = 0.01 // Cost factor per input token
	MergeFragmentsBaseCost = 0.5  // Base cost for merging
	MergeFragmentsPerField = 0.1  // Additional cost per field
	TransformCost          = 1.5  // Cost for transformations
	DefaultNodeCost        = 1.0  // Default cost for unknown types
)

// Gemini model pricing (per 1M tokens)
const (
	GeminiFlashInputCost  = 0.075 // $0.075 per 1M input tokens (Gemini 1.5 Flash)
	GeminiFlashOutputCost = 0.30  // $0.30 per 1M output tokens (Gemini 1.5 Flash)
	GeminiProInputCost    = 1.25  // $1.25 per 1M input tokens (Gemini 1.5 Pro)
	GeminiProOutputCost   = 5.00  // $5.00 per 1M output tokens (Gemini 1.5 Pro)
)

// DefaultModelPricing returns current input/output token costs (USD / 1K tokens).
// DefaultModelPricing returns current input/output token costs (USD per 1 K tokens).
func DefaultModelPricing() map[string]ModelPrice {
	return map[string]ModelPrice{
		// OpenAI
		"gpt-4o":        {PromptTokCost: 0.0050, CompletionTokCost: 0.0200}, // $5 / M in,  $20 / M out  (OpenAI pricing)
		"gpt-4o-mini":   {PromptTokCost: 0.0006, CompletionTokCost: 0.0024}, // $0.60 / M in, $2.40 / M out
		"gpt-4.1":       {PromptTokCost: 0.0020, CompletionTokCost: 0.0080}, // $2 / M in,  $8 / M out
		"gpt-4.1-mini":  {PromptTokCost: 0.0004, CompletionTokCost: 0.0016}, // $0.40 / M in, $1.60 / M out
		"gpt-4.1-nano":  {PromptTokCost: 0.0001, CompletionTokCost: 0.0004}, // $0.10 / M in, $0.40 / M out
		"gpt-3.5-turbo": {PromptTokCost: 0.0005, CompletionTokCost: 0.0015}, // $0.50 / M in, $1.50 / M out

		// Google Gemini
		"gemini-2.5-pro":   {PromptTokCost: 0.00125, CompletionTokCost: 0.0100},   // $1.25 / M in, $10 / M out  (Vertex AI pricing)
		"gemini-2.5-flash": {PromptTokCost: 0.00030, CompletionTokCost: 0.0025},   // $0.30 / M in, $2.50 / M out
		"gemini-2.0-flash": {PromptTokCost: 0.00015, CompletionTokCost: 0.0006},   // $0.15 / M in, $0.60 / M out
		"gemini-1.5-pro":   {PromptTokCost: 0.00125, CompletionTokCost: 0.0050},   // $1.25 / M in,  $5 / M out
		"gemini-1.5-flash": {PromptTokCost: 0.000075, CompletionTokCost: 0.00030}, // $0.075 / M in, $0.30 / M out

		// Anthropic Claude 3
		"claude-3-opus":   {PromptTokCost: 0.0150, CompletionTokCost: 0.0750}, // $15 / M in, $75 / M out  (Anthropic pricing)
		"claude-3-sonnet": {PromptTokCost: 0.0030, CompletionTokCost: 0.0150}, // $3 / M in,  $15 / M out
		"claude-3-haiku":  {PromptTokCost: 0.0008, CompletionTokCost: 0.0040}, // $0.80 / M in, $4 / M out
	}
}

// Field type categories for token estimation
type FieldCategory int

const (
	SimpleField  FieldCategory = iota // Simple fields like name, age
	MediumField                       // Medium complexity like email, phone
	ComplexField                      // Complex fields like address, description
)

// Field type configuration
var fieldTypeConfig = map[string]struct {
	category     FieldCategory
	inputTokens  int
	outputTokens int
}{
	"name":        {SimpleField, 100, 20},
	"age":         {SimpleField, 80, 10},
	"email":       {MediumField, 90, 25},
	"phone":       {MediumField, 85, 15},
	"address":     {ComplexField, 150, 40},
	"description": {ComplexField, 200, 60},
	"title":       {MediumField, 120, 30},
	"company":     {MediumField, 110, 25},
	"url":         {MediumField, 95, 20},
	"date":        {SimpleField, 85, 12},
}

// Default token estimates for unknown field types
const (
	DefaultInputTokens  = 100
	DefaultOutputTokens = 30
	DefaultDryRunModel  = "gpt-3.5-turbo" // Default model for dry run operations
)

// Helper functions for common operations

// getFieldTokenEstimates returns input and output token estimates for a field
func getFieldTokenEstimates(fieldName string) (int, int) {
	fieldKey := strings.ToLower(fieldName)
	if config, exists := fieldTypeConfig[fieldKey]; exists {
		return config.inputTokens, config.outputTokens
	}
	return DefaultInputTokens, DefaultOutputTokens
}

// extractUniqueStrings extracts unique strings from a slice
func extractUniqueStrings(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// calculateTokenCost calculates cost based on input/output tokens and pricing
func calculateTokenCost(inputTokens, outputTokens int, price ModelPrice) float64 {
	inputCost := float64(inputTokens) * price.PromptTokCost / 1000.0
	outputCost := float64(outputTokens) * price.CompletionTokCost / 1000.0
	return inputCost + outputCost
}

// DryRunner interface for types that can perform dry-run execution
type DryRunner interface {
	DryRun(ctx context.Context, assets []Asset, optFns ...func(*Options)) (*ExecutionStats, error)
}

// ExecutionStats tracks actual execution statistics for comparison with planned execution.
type ExecutionStats struct {
	PromptCalls       int              `json:"promptCalls"`       // Total number of prompt calls made
	ModelCalls        map[string]int   `json:"modelCalls"`        // Number of calls per model
	PromptGroups      int              `json:"promptGroups"`      // Number of distinct prompt groups
	FieldsExtracted   int              `json:"fieldsExtracted"`   // Total number of fields processed
	GroupDetails      []GroupExecution `json:"groupDetails"`      // Details of each group execution
	TotalInputTokens  int              `json:"totalInputTokens"`  // Total input tokens (estimated)
	TotalOutputTokens int              `json:"totalOutputTokens"` // Total output tokens (estimated)
}

// GroupExecution represents statistics for a single prompt group execution.
type GroupExecution struct {
	PromptName   string   `json:"promptName"`   // Name/key of the prompt
	Model        string   `json:"model"`        // Model used for this group
	Fields       []string `json:"fields"`       // Fields processed by this group
	InputTokens  int      `json:"inputTokens"`  // Estimated input tokens
	OutputTokens int      `json:"outputTokens"` // Estimated output tokens
	ParentPath   string   `json:"parentPath"`   // Parent path for nested structures
}

// PlanNodeType defines the type of operation a node represents.
type PlanNodeType string

const (
	SchemaAnalysisType PlanNodeType = "SchemaAnalysis"
	PromptCallType     PlanNodeType = "PromptCall"
	MergeFragmentsType PlanNodeType = "MergeFragments"
	TransformType      PlanNodeType = "Transform"
)

// PlanNode represents a node in the Unstructor execution plan.
// Warning: Children and Metadata are exported for extensibility but should not be
// modified after plan generation to maintain internal consistency.
type PlanNode struct {
	Type         PlanNodeType           `json:"type"`                   // e.g. "SchemaAnalysis", "PromptCall", ...
	PromptName   string                 `json:"promptName,omitempty"`   // Name/identifier of the prompt (if applicable)
	Model        string                 `json:"model,omitempty"`        // LLM model used (if applicable)
	Fields       []string               `json:"fields,omitempty"`       // Fields covered/extracted at this node
	InputTokens  int                    `json:"inputTokens,omitempty"`  // Estimated input size in tokens for this node
	OutputTokens int                    `json:"outputTokens,omitempty"` // Estimated output size in tokens for this node
	EstCost      float64                `json:"estCost"`                // Estimated *abstract* cost units for this node (includes children)
	ActCost      *float64               `json:"actCost,omitempty"`      // Optional actual cost (in $ or token units) if calculated
	Children     []*PlanNode            `json:"children,omitempty"`     // Child plan nodes (sub-operations)
	Metadata     map[string]interface{} `json:"metadata,omitempty"`     // Additional metadata for extensibility
	// Summary information (populated for root nodes)
	ExpectedModels     []string       `json:"expectedModels,omitempty"`     // Models expected to be used in this plan
	ExpectedCallCounts map[string]int `json:"expectedCallCounts,omitempty"` // Expected prompt call counts by model
}

// ModelPrice represents the pricing for a specific model.
type ModelPrice struct {
	PromptTokCost     float64 // Cost per 1000 input tokens
	CompletionTokCost float64 // Cost per 1000 output tokens
}

// ExplainOptions contains options for plan explanation.
type ExplainOptions struct {
	IncludeActualCosts   bool
	ModelPrices          map[string]ModelPrice
	EstimateOutputTokens bool
}

// FormatType represents different output formats for the execution plan.
type FormatType string

const (
	FormatText FormatType = "text"
	FormatJSON FormatType = "json"
)

// PlanBuilder is responsible for constructing execution plans.
// Note: PlanBuilder is not thread-safe. Create separate instances for concurrent use.
type PlanBuilder struct {
	schema       interface{}
	promptConfig map[string]interface{}
	modelConfig  map[string]string
	unstructor   interface{} // Generic unstructor for dry-run execution
	document     string      // Sample document for token estimation
}

// NewPlanBuilder creates a new plan builder.
func NewPlanBuilder() *PlanBuilder {
	return &PlanBuilder{
		promptConfig: make(map[string]interface{}),
		modelConfig:  make(map[string]string),
	}
}

// WithSchema sets the schema for the plan.
func (pb *PlanBuilder) WithSchema(schema interface{}) *PlanBuilder {
	pb.schema = schema
	return pb
}

// WithPromptConfig sets the prompt configuration.
func (pb *PlanBuilder) WithPromptConfig(config map[string]interface{}) *PlanBuilder {
	pb.promptConfig = config
	return pb
}

// WithModelConfig sets the model configuration.
func (pb *PlanBuilder) WithModelConfig(config map[string]string) *PlanBuilder {
	pb.modelConfig = config
	return pb
}

// WithUnstructor sets an Unstructor instance for dry-run execution.
// This enables more accurate plan generation using actual execution flow.
func (pb *PlanBuilder) WithUnstructor(unstructor interface{}) *PlanBuilder {
	pb.unstructor = unstructor
	return pb
}

// WithSampleDocument sets a sample document for token estimation.
// This is used with WithUnstructor for more accurate token counts.
func (pb *PlanBuilder) WithSampleDocument(doc string) *PlanBuilder {
	pb.document = doc
	return pb
}

// Explain generates the execution plan with abstract cost estimates.
func (pb *PlanBuilder) Explain() (*PlanNode, error) {
	return pb.buildPlan(ExplainOptions{})
}

// ExplainWithCosts generates the execution plan with real cost estimates.
func (pb *PlanBuilder) ExplainWithCosts(pricing map[string]ModelPrice) (*PlanNode, error) {
	options := ExplainOptions{
		IncludeActualCosts:   true,
		ModelPrices:          pricing,
		EstimateOutputTokens: true,
	}
	return pb.buildPlan(options)
}

// buildPlan constructs the execution plan based on the provided options with clear fallback logic.
func (pb *PlanBuilder) buildPlan(options ExplainOptions) (*PlanNode, error) {
	if pb.schema == nil {
		return nil, fmt.Errorf("schema is required to build execution plan")
	}

	// Prefer dry-run execution if available, otherwise use static analysis
	if pb.canPerformDryRun() {
		if plan, err := pb.buildPlanFromDryRun(options); err == nil {
			return plan, nil
		}
		// Log error but continue with static analysis
	}

	return pb.buildPlanFromStaticAnalysis(options)
}

// canPerformDryRun checks if dry-run execution is possible
func (pb *PlanBuilder) canPerformDryRun() bool {
	return pb.unstructor != nil && pb.document != ""
}

// buildPlanFromDryRun constructs the plan using actual dry-run execution.
func (pb *PlanBuilder) buildPlanFromDryRun(options ExplainOptions) (*PlanNode, error) {
	// This method would need type assertion for the specific Unstructor type
	// For now, we'll implement a generic interface approach

	// Try to call DryRun method via reflection or interface
	stats, err := pb.callDryRun()
	if err != nil {
		// Fall back to static analysis if dry-run fails
		return pb.buildPlanFromStaticAnalysis(options)
	}

	// Create root SchemaAnalysis node
	rootNode := &PlanNode{
		Type:        SchemaAnalysisType,
		Fields:      pb.getFieldsFromStats(stats),
		InputTokens: 10, // Schema analysis overhead
		Children:    make([]*PlanNode, 0),
		Metadata:    make(map[string]interface{}),
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
			Children:     make([]*PlanNode, 0),
			Metadata:     make(map[string]interface{}),
		}

		// Calculate cost if pricing is available
		if options.IncludeActualCosts && options.ModelPrices != nil {
			actualCost := pb.calculateActualCost(promptNode, options.ModelPrices)
			if actualCost > 0 {
				promptNode.ActCost = &actualCost
			}
		}

		rootNode.Children = append(rootNode.Children, promptNode)
	}

	// Create MergeFragments node
	mergeNode := &PlanNode{
		Type:     MergeFragmentsType,
		Fields:   pb.getFieldsFromStats(stats),
		Children: make([]*PlanNode, 0),
		Metadata: make(map[string]interface{}),
	}
	rootNode.Children = append(rootNode.Children, mergeNode)

	// Calculate costs
	pb.calculateCosts(rootNode, options)

	// Populate summary information from dry-run stats
	rootNode.ExpectedModels = pb.getModelsFromStats(stats)
	rootNode.ExpectedCallCounts = stats.ModelCalls

	return rootNode, nil
}

// callDryRun attempts to call DryRun on the configured Unstructor with simplified error handling.
func (pb *PlanBuilder) callDryRun() (*ExecutionStats, error) {
	if pb.unstructor == nil {
		return nil, fmt.Errorf("unstructor not configured")
	}
	if pb.document == "" {
		return nil, fmt.Errorf("sample document not configured")
	}

	dryRunner, ok := pb.unstructor.(DryRunner)
	if !ok {
		return nil, fmt.Errorf("unstructor does not implement DryRunner interface")
	}

	assets := []Asset{&TextAsset{Content: pb.document}}
	return dryRunner.DryRun(context.Background(), assets, WithModel(DefaultDryRunModel))
}

// getFieldsFromStats extracts all unique field names from execution statistics.
func (pb *PlanBuilder) getFieldsFromStats(stats *ExecutionStats) []string {
	var allFields []string
	for _, group := range stats.GroupDetails {
		allFields = append(allFields, group.Fields...)
	}
	return extractUniqueStrings(allFields)
}

// getModelsFromStats extracts unique model names from execution statistics.
func (pb *PlanBuilder) getModelsFromStats(stats *ExecutionStats) []string {
	var models []string
	for model := range stats.ModelCalls {
		models = append(models, model)
	}
	return models
}

// buildPlanFromStaticAnalysis constructs the plan using static schema analysis.
func (pb *PlanBuilder) buildPlanFromStaticAnalysis(options ExplainOptions) (*PlanNode, error) {

	// Extract fields from schema (simplified - this would be more complex in reality)
	fields, err := pb.extractFieldsFromSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to extract fields from schema: %w", err)
	}

	// Create root SchemaAnalysis node
	rootNode := &PlanNode{
		Type:        SchemaAnalysisType,
		Fields:      fields,
		InputTokens: pb.estimateSchemaTokens(fields),
		Children:    make([]*PlanNode, 0),
		Metadata:    make(map[string]interface{}),
	}

	// Create PromptCall nodes for each field
	for _, field := range fields {
		promptNode := pb.createPromptCallNode(field, options)
		rootNode.Children = append(rootNode.Children, promptNode)
	}

	// Create MergeFragments node
	mergeNode := &PlanNode{
		Type:        MergeFragmentsType,
		Fields:      fields,
		InputTokens: 0, // Merge doesn't typically use tokens
		Children:    make([]*PlanNode, 0),
		Metadata:    make(map[string]interface{}),
	}

	// Add merge node as final child
	rootNode.Children = append(rootNode.Children, mergeNode)

	// Calculate costs
	pb.calculateCosts(rootNode, options)

	// Populate summary information (expected models and call counts)
	pb.populateSummaryInfo(rootNode)

	return rootNode, nil
}

// extractFieldsFromSchema extracts field names from the schema with improved error handling.
func (pb *PlanBuilder) extractFieldsFromSchema() ([]string, error) {
	schemaMap, ok := pb.schema.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema must be a map, got %T", pb.schema)
	}

	fields, exists := schemaMap["fields"]
	if !exists {
		return nil, fmt.Errorf("schema missing required 'fields' key")
	}

	return pb.parseFieldsFromInterface(fields)
}

// parseFieldsFromInterface converts various field representations to string slice
func (pb *PlanBuilder) parseFieldsFromInterface(fields interface{}) ([]string, error) {
	switch v := fields.(type) {
	case []string:
		return pb.validateFields(v)
	case []interface{}:
		return pb.convertInterfaceSliceToStrings(v)
	default:
		return nil, fmt.Errorf("schema 'fields' must be a string array, got %T", fields)
	}
}

// convertInterfaceSliceToStrings converts []interface{} to []string with validation
func (pb *PlanBuilder) convertInterfaceSliceToStrings(fields []interface{}) ([]string, error) {
	result := make([]string, len(fields))
	for i, field := range fields {
		fieldStr, ok := field.(string)
		if !ok {
			return nil, fmt.Errorf("field at index %d must be string, got %T", i, field)
		}
		result[i] = fieldStr
	}
	return pb.validateFields(result)
}

// validateFields ensures the field list is not empty
func (pb *PlanBuilder) validateFields(fields []string) ([]string, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("schema cannot have empty 'fields' array")
	}
	return fields, nil
}

// estimateSchemaTokens estimates the token count for schema analysis using constants.
func (pb *PlanBuilder) estimateSchemaTokens(fields []string) int {
	return SchemaBaseTokens + len(fields)*TokensPerField
}

// createPromptCallNode creates a PromptCall node for a specific field.
func (pb *PlanBuilder) createPromptCallNode(field string, options ExplainOptions) *PlanNode {
	model := pb.getModelForField(field)
	promptName := pb.getPromptNameForField(field)

	inputTokens := pb.estimatePromptTokens(field)
	outputTokens := 0
	if options.EstimateOutputTokens {
		outputTokens = pb.estimateOutputTokens(field)
	}

	node := &PlanNode{
		Type:         PromptCallType,
		PromptName:   promptName,
		Model:        model,
		Fields:       []string{field},
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Children:     make([]*PlanNode, 0),
		Metadata:     make(map[string]interface{}),
	}

	return node
}

// getPromptNameForField returns the prompt name for a specific field, using promptConfig if available.
func (pb *PlanBuilder) getPromptNameForField(field string) string {
	if pb.promptConfig != nil {
		if promptName, exists := pb.promptConfig[field]; exists {
			if name, ok := promptName.(string); ok {
				return name
			}
		}
	}
	// Default prompt name
	return fmt.Sprintf("%sExtractionPrompt", field)
}

// getModelForField returns the model to use for a specific field with fallback to default.
func (pb *PlanBuilder) getModelForField(field string) string {
	if model, exists := pb.modelConfig[field]; exists {
		return model
	}
	return DefaultDryRunModel
}

// estimatePromptTokens estimates the input token count for a prompt using field configuration.
func (pb *PlanBuilder) estimatePromptTokens(field string) int {
	fieldTokens, _ := getFieldTokenEstimates(field)
	return BasePromptTokens + fieldTokens + DocumentTokens
}

// estimateOutputTokens estimates the output token count for a prompt using field configuration.
func (pb *PlanBuilder) estimateOutputTokens(field string) int {
	_, outputTokens := getFieldTokenEstimates(field)
	return outputTokens
}

// calculateCosts calculates abstract and actual costs for all nodes.
func (pb *PlanBuilder) calculateCosts(node *PlanNode, options ExplainOptions) {
	// Calculate costs for children first (bottom-up)
	childrenCost := 0.0
	for _, child := range node.Children {
		pb.calculateCosts(child, options)
		childrenCost += child.EstCost
	}

	// Calculate node's own cost
	nodeCost := pb.calculateNodeCost(node)

	// Total estimated cost includes children
	node.EstCost = nodeCost + childrenCost

	// Calculate actual cost if requested
	if options.IncludeActualCosts {
		actualCost := pb.calculateActualCost(node, options.ModelPrices)
		if actualCost > 0 {
			node.ActCost = &actualCost
		}
	}
}

// populateSummaryInfo collects expected models and call counts from the plan tree.
func (pb *PlanBuilder) populateSummaryInfo(rootNode *PlanNode) {
	models := make(map[string]bool)
	callCounts := make(map[string]int)

	var collectStats func(*PlanNode)
	collectStats = func(node *PlanNode) {
		if node.Type == PromptCallType && node.Model != "" {
			models[node.Model] = true
			callCounts[node.Model]++
		}
		for _, child := range node.Children {
			collectStats(child)
		}
	}

	collectStats(rootNode)

	// Convert models map to slice
	expectedModels := make([]string, 0, len(models))
	for model := range models {
		expectedModels = append(expectedModels, model)
	}

	// Only populate summary info on root node
	rootNode.ExpectedModels = expectedModels
	rootNode.ExpectedCallCounts = callCounts
}

// calculateNodeCost calculates the abstract cost for a single node using constants.
func (pb *PlanBuilder) calculateNodeCost(node *PlanNode) float64 {
	switch node.Type {
	case SchemaAnalysisType:
		return SchemaAnalysisBaseCost + float64(len(node.Fields))*SchemaAnalysisPerField
	case PromptCallType:
		return PromptCallBaseCost + float64(node.InputTokens)*PromptCallTokenFactor
	case MergeFragmentsType:
		return MergeFragmentsBaseCost + float64(len(node.Fields))*MergeFragmentsPerField
	case TransformType:
		return TransformCost
	default:
		return DefaultNodeCost
	}
}

// calculateActualCost calculates the real cost in USD for a node using helper function.
func (pb *PlanBuilder) calculateActualCost(node *PlanNode, pricing map[string]ModelPrice) float64 {
	if node.Type != PromptCallType || node.Model == "" || pricing == nil {
		return 0.0
	}

	price, exists := pricing[node.Model]
	if !exists {
		return 0.0
	}

	outputTokens := node.OutputTokens
	if outputTokens == 0 && len(node.Fields) > 0 {
		outputTokens = pb.estimateOutputTokens(node.Fields[0])
	}

	return calculateTokenCost(node.InputTokens, outputTokens, price)
}

// ExplainPretty returns a human-readable formatted plan.
func (pb *PlanBuilder) ExplainPretty(format FormatType) (string, error) {
	plan, err := pb.Explain()
	if err != nil {
		return "", err
	}

	return pb.FormatPlan(plan, format)
}

// ExplainPrettyWithCosts returns a human-readable formatted plan with costs.
func (pb *PlanBuilder) ExplainPrettyWithCosts(format FormatType, pricing map[string]ModelPrice) (string, error) {
	if pricing == nil {
		return "", fmt.Errorf("pricing information is required for cost calculations")
	}

	plan, err := pb.ExplainWithCosts(pricing)
	if err != nil {
		return "", err
	}

	return pb.FormatPlan(plan, format)
}

// FormatPlan formats a plan according to the specified format.
func (pb *PlanBuilder) FormatPlan(plan *PlanNode, format FormatType) (string, error) {
	switch format {
	case FormatText:
		return pb.formatAsText(plan), nil
	case FormatJSON:
		return pb.formatAsJSON(plan)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// EstimateTokensFromText provides a rough token estimate from text length using constants.
func EstimateTokensFromText(text string) int {
	return (len(text) + CharsPerToken - 1) / CharsPerToken
}

// EstimateTokensFromWords provides a rough token estimate from word count using constants.
func EstimateTokensFromWords(wordCount int) int {
	ratio := int(TokensPerWordRatio * 10) // Convert 1.3 to 13 for integer math
	return (wordCount*ratio + 9) / 10
}
