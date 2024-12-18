package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	operators_db "github.com/vMaroon/Kuery/pkg/operators-db"
)

// OperatorsTool is a tool that retrieves the operator schema that is most relevant to the prompt.
type OperatorsTool struct {
	operators_db.OperatorsRetriever
}

func (ot *OperatorsTool) Name() string {
	return "retrieveOperatorByRelevance"
}

func (ot *OperatorsTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name:        "retrieveOperatorByRelevance",
			Description: "Retrieve the operator schema that is most relevant to the prompt",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "The prompt to retrieve the operator schema for",
					},
				},
				"required": []string{"prompt"},
			},
		},
	}
}

func (ot *OperatorsTool) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	schema, err := ot.RetrieveOperator(ctx, args.Prompt)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to get current weather: %v", err),
		}
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    schema.Name,
	}
}
