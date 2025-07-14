package unstruct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

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
	FormatText     FormatType = "text"
	FormatJSON     FormatType = "json"
	FormatGraphviz FormatType = "dot"
	FormatHTML     FormatType = "html"
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

// buildPlan constructs the execution plan based on the provided options.
func (pb *PlanBuilder) buildPlan(options ExplainOptions) (*PlanNode, error) {
	if pb.schema == nil {
		return nil, fmt.Errorf("schema is required to build execution plan")
	}

	// Try dry-run execution if Unstructor is available
	if pb.unstructor != nil && pb.document != "" {
		return pb.buildPlanFromDryRun(options)
	}

	// Fall back to static analysis
	return pb.buildPlanFromStaticAnalysis(options)
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

// callDryRun attempts to call DryRun on the configured Unstructor
func (pb *PlanBuilder) callDryRun() (*ExecutionStats, error) {
	if pb.unstructor == nil || pb.document == "" {
		return nil, fmt.Errorf("unstructor or sample document not configured")
	}

	// Check if the unstructor implements DryRunner interface
	dryRunner, ok := pb.unstructor.(DryRunner)
	if !ok {
		return nil, fmt.Errorf("unstructor does not implement DryRunner interface")
	}

	// Call DryRun with a default configuration
	assets := []Asset{&TextAsset{Content: pb.document}}
	return dryRunner.DryRun(context.Background(), assets, WithModel("gpt-3.5-turbo"))
}

// getFieldsFromStats extracts all field names from execution statistics
func (pb *PlanBuilder) getFieldsFromStats(stats *ExecutionStats) []string {
	fieldSet := make(map[string]bool)
	for _, group := range stats.GroupDetails {
		for _, field := range group.Fields {
			fieldSet[field] = true
		}
	}

	fields := make([]string, 0, len(fieldSet))
	for field := range fieldSet {
		fields = append(fields, field)
	}
	return fields
}

// getModelsFromStats extracts unique model names from execution statistics
func (pb *PlanBuilder) getModelsFromStats(stats *ExecutionStats) []string {
	models := make([]string, 0, len(stats.ModelCalls))
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

// extractFieldsFromSchema extracts field names from the schema.
func (pb *PlanBuilder) extractFieldsFromSchema() ([]string, error) {
	// Try to extract fields from the schema map
	if schemaMap, ok := pb.schema.(map[string]interface{}); ok {
		if fields, exists := schemaMap["fields"]; exists {
			if fieldSlice, ok := fields.([]string); ok {
				if len(fieldSlice) == 0 {
					return nil, fmt.Errorf("schema contains empty 'fields' array")
				}
				return fieldSlice, nil
			}
			// Handle []interface{} case
			if fieldInterface, ok := fields.([]interface{}); ok {
				result := make([]string, len(fieldInterface))
				for i, field := range fieldInterface {
					if fieldStr, ok := field.(string); ok {
						result[i] = fieldStr
					} else {
						return nil, fmt.Errorf("schema field at index %d is not a string", i)
					}
				}
				if len(result) == 0 {
					return nil, fmt.Errorf("schema contains empty 'fields' array")
				}
				return result, nil
			}
			return nil, fmt.Errorf("schema 'fields' is not a string array")
		}
		return nil, fmt.Errorf("schema missing 'fields' key")
	}
	return nil, fmt.Errorf("schema is not a map or unsupported type")
}

// estimateSchemaTokens estimates the token count for schema analysis.
func (pb *PlanBuilder) estimateSchemaTokens(fields []string) int {
	// Base overhead for schema analysis
	baseTokens := 20

	// Add tokens per field (field name + overhead)
	tokensPerField := 5

	return baseTokens + len(fields)*tokensPerField
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

// getModelForField returns the model to use for a specific field.
func (pb *PlanBuilder) getModelForField(field string) string {
	if model, exists := pb.modelConfig[field]; exists {
		return model
	}
	// Default model
	return "gpt-3.5-turbo"
}

// estimatePromptTokens estimates the input token count for a prompt.
func (pb *PlanBuilder) estimatePromptTokens(field string) int {
	// Base prompt template tokens
	baseTokens := 50

	// Field-specific content tokens (estimated based on field type)
	var fieldTokens int
	switch strings.ToLower(field) {
	case "name":
		fieldTokens = 100
	case "age":
		fieldTokens = 80
	case "address":
		fieldTokens = 150
	case "email":
		fieldTokens = 90
	default:
		fieldTokens = 100
	}

	// Document context tokens (assuming we include relevant context)
	contextTokens := 200

	return baseTokens + fieldTokens + contextTokens
}

// estimateOutputTokens estimates the output token count for a prompt.
func (pb *PlanBuilder) estimateOutputTokens(field string) int {
	// Estimate based on typical response length for field type
	switch strings.ToLower(field) {
	case "name":
		return 20
	case "age":
		return 10
	case "address":
		return 40
	case "email":
		return 25
	default:
		return 30
	}
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

// calculateNodeCost calculates the abstract cost for a single node.
func (pb *PlanBuilder) calculateNodeCost(node *PlanNode) float64 {
	switch node.Type {
	case SchemaAnalysisType:
		// Cost proportional to number of fields
		return 1.0 + float64(len(node.Fields))*0.5
	case PromptCallType:
		// Base cost plus token-based cost
		baseCost := 3.0
		tokenCost := float64(node.InputTokens) * 0.01
		return baseCost + tokenCost
	case MergeFragmentsType:
		// Small constant cost
		return 0.5 + float64(len(node.Fields))*0.1
	case TransformType:
		// Medium cost for transformations
		return 1.5
	default:
		return 1.0
	}
}

// calculateActualCost calculates the real cost in USD for a node.
func (pb *PlanBuilder) calculateActualCost(node *PlanNode, pricing map[string]ModelPrice) float64 {
	if node.Type != PromptCallType || node.Model == "" {
		return 0.0
	}

	if pricing == nil {
		return 0.0
	}

	price, exists := pricing[node.Model]
	if !exists {
		return 0.0
	}

	inputCost := float64(node.InputTokens) * price.PromptTokCost / 1000.0

	// Ensure we have output tokens for cost calculation
	outputTokens := node.OutputTokens
	if outputTokens == 0 && len(node.Fields) > 0 {
		outputTokens = pb.estimateOutputTokens(node.Fields[0])
	}

	outputCost := float64(outputTokens) * price.CompletionTokCost / 1000.0

	return inputCost + outputCost
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
	case FormatGraphviz:
		return pb.formatAsGraphviz(plan), nil
	case FormatHTML:
		return pb.formatAsHTML(plan), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// formatAsText formats the plan as an ASCII tree.
func (pb *PlanBuilder) formatAsText(plan *PlanNode) string {
	var sb strings.Builder
	sb.WriteString("Unstructor Execution Plan (estimated costs)\n")
	pb.formatNodeAsText(plan, "", true, &sb)
	return sb.String()
}

// formatNodeAsText recursively formats a node and its children as text.
func (pb *PlanBuilder) formatNodeAsText(node *PlanNode, prefix string, isLast bool, sb *strings.Builder) {
	// Choose the appropriate tree connector
	connector := "├─ "
	if isLast {
		connector = "└─ "
	}
	if prefix == "" {
		connector = ""
	}

	// Format node information
	nodeStr := pb.formatNodeInfo(node)
	sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, nodeStr))

	// Format children
	childPrefix := prefix
	if prefix == "" {
		// First level children get "  " as prefix to properly indent them
		childPrefix = "  "
	} else {
		if isLast {
			childPrefix += "   "
		} else {
			childPrefix += "│  "
		}
	}

	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		pb.formatNodeAsText(child, childPrefix, isLastChild, sb)
	}
}

// formatNodeInfo formats information for a single node.
func (pb *PlanBuilder) formatNodeInfo(node *PlanNode) string {
	parts := []string{string(node.Type)}

	if node.PromptName != "" {
		parts = append(parts, fmt.Sprintf(`"%s"`, node.PromptName))
	}

	var details []string

	if node.Model != "" {
		details = append(details, fmt.Sprintf("model=%s", node.Model))
	}

	details = append(details, fmt.Sprintf("cost=%.1f", node.EstCost))

	// Display token information more clearly
	if node.InputTokens > 0 || node.OutputTokens > 0 {
		if node.OutputTokens > 0 {
			details = append(details, fmt.Sprintf("tokens(in=%d,out=%d)", node.InputTokens, node.OutputTokens))
		} else {
			details = append(details, fmt.Sprintf("tokens(in=%d)", node.InputTokens))
		}
	}

	if len(node.Fields) > 0 {
		if len(node.Fields) == 1 {
			details = append(details, fmt.Sprintf("field=%s", node.Fields[0]))
		} else {
			details = append(details, fmt.Sprintf("fields=%v", node.Fields))
		}
	}

	if node.ActCost != nil {
		details = append(details, fmt.Sprintf("$%.6f", *node.ActCost))
	}

	if len(details) > 0 {
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(details, ", ")))
	}

	return strings.Join(parts, " ")
}

// formatAsJSON formats the plan as JSON.
func (pb *PlanBuilder) formatAsJSON(plan *PlanNode) (string, error) {
	bytes, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// formatAsGraphviz formats the plan as Graphviz DOT format.
func (pb *PlanBuilder) formatAsGraphviz(plan *PlanNode) string {
	var sb strings.Builder
	sb.WriteString("digraph UnstructorPlan {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n")

	nodeCounter := 0
	nodeMap := make(map[*PlanNode]string)

	// Generate nodes and edges
	pb.generateGraphvizNodes(plan, &nodeCounter, nodeMap, &sb)
	pb.generateGraphvizEdges(plan, nodeMap, &sb)

	sb.WriteString("}\n")
	return sb.String()
}

// generateGraphvizNodes generates Graphviz nodes recursively.
func (pb *PlanBuilder) generateGraphvizNodes(node *PlanNode, counter *int, nodeMap map[*PlanNode]string, sb *strings.Builder) {
	nodeID := fmt.Sprintf("node%d", *counter)
	*counter++
	nodeMap[node] = nodeID

	label := pb.formatGraphvizNodeLabel(node)
	sb.WriteString(fmt.Sprintf("  %s [label=\"%s\"];\n", nodeID, label))

	for _, child := range node.Children {
		pb.generateGraphvizNodes(child, counter, nodeMap, sb)
	}
}

// generateGraphvizEdges generates Graphviz edges.
func (pb *PlanBuilder) generateGraphvizEdges(node *PlanNode, nodeMap map[*PlanNode]string, sb *strings.Builder) {
	nodeID := nodeMap[node]

	for _, child := range node.Children {
		childID := nodeMap[child]
		sb.WriteString(fmt.Sprintf("  %s -> %s;\n", nodeID, childID))
		pb.generateGraphvizEdges(child, nodeMap, sb)
	}
}

// formatGraphvizNodeLabel formats a node label for Graphviz.
func (pb *PlanBuilder) formatGraphvizNodeLabel(node *PlanNode) string {
	var parts []string

	if node.PromptName != "" {
		// Escape quotes and backslashes for Graphviz
		escapedPrompt := strings.ReplaceAll(strings.ReplaceAll(node.PromptName, `\`, `\\`), `"`, `\"`)
		parts = append(parts, fmt.Sprintf("%s: %s", node.Type, escapedPrompt))
	} else {
		parts = append(parts, string(node.Type))
	}

	if node.Model != "" {
		// Escape quotes and backslashes for Graphviz
		escapedModel := strings.ReplaceAll(strings.ReplaceAll(node.Model, `\`, `\\`), `"`, `\"`)
		parts = append(parts, fmt.Sprintf("model: %s", escapedModel))
	}

	parts = append(parts, fmt.Sprintf("cost=%.1f", node.EstCost))

	if len(node.Fields) > 0 && len(node.Fields) <= 2 {
		// Escape field names as well
		escapedFields := make([]string, len(node.Fields))
		for i, field := range node.Fields {
			escapedFields[i] = strings.ReplaceAll(strings.ReplaceAll(field, `\`, `\\`), `"`, `\"`)
		}
		parts = append(parts, fmt.Sprintf("fields: %s", strings.Join(escapedFields, ", ")))
	} else if len(node.Fields) > 2 {
		parts = append(parts, fmt.Sprintf("fields: %d", len(node.Fields)))
	}

	return strings.Join(parts, "\\n")
}

// formatAsHTML formats the plan as HTML.
func (pb *PlanBuilder) formatAsHTML(plan *PlanNode) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
    <title>Unstructor Execution Plan</title>
    <style>
        body { font-family: monospace; margin: 20px; }
        .plan-tree { background: #f5f5f5; padding: 15px; border-radius: 5px; }
        .node { margin: 2px 0; }
        .node-type { font-weight: bold; color: #0066cc; }
        .node-details { color: #666; }
        .indent { margin-left: 20px; }
    </style>
</head>
<body>
    <h1>Unstructor Execution Plan</h1>
    <div class="plan-tree">
        <pre>`)

	sb.WriteString(pb.formatAsText(plan))

	sb.WriteString(`</pre>
    </div>
</body>
</html>`)

	return sb.String()
}

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

// EstimateTokensFromText provides a rough token estimate from text length.
func EstimateTokensFromText(text string) int {
	// Rough heuristic: ~4 characters per token for English text
	return (len(text) + 3) / 4
}

// EstimateTokensFromWords provides a rough token estimate from word count.
func EstimateTokensFromWords(wordCount int) int {
	// Rough heuristic: ~1.3 tokens per word
	return (wordCount*13 + 9) / 10
}
