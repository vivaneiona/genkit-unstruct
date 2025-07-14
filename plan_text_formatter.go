package unstruct

import (
	"fmt"
	"strings"
)

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
