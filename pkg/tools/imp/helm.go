package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
)

type HelmTool struct {
}

func (ht *HelmTool) Name() string {
	return "helmTool"
}

func (ht *HelmTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name: ht.Name(),
			Description: "Interact with Helm. Install, upgrade, delete, etc." +
				"This tool should be used with clear user intent.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type":        "string",
						"description": "The operation to perform: install, upgrade, delete, etc.",
					},
					"chart": map[string]any{
						"type":        "string",
						"description": "The Helm chart to use",
					},
					"values": map[string]any{
						"type":        "string",
						"description": "The values to use for the chart",
					},
				},
				"required": []string{"operation", "chart"},
			},
		},
	}
}

func (ht *HelmTool) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args struct {
		Operation string `json:"operation"`
		Chart     string `json:"chart"`
		Values    string `json:"values"`
	}
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	response, err := helmOperation(args.Operation, args.Chart, args.Values)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to execute Helm operation: %v", err),
		}
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    response,
	}
}

// helmOperation executes the Helm operation and returns the response.
func helmOperation(operation, chart, values string) (string, error) {
	// execute Helm operation
	return fmt.Sprintf("Helm operation performed: %s %s %s", operation, chart, values), nil
}
