package unstruct

import (
	"encoding/json"
)

// formatAsJSON formats the plan as JSON.
func (pb *PlanBuilder) formatAsJSON(plan *PlanNode) (string, error) {
	bytes, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
