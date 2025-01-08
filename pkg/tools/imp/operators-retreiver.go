package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	operators_db "github.com/vMaroon/Kuery/pkg/operators-db"
)

// OperatorsRAGTool is a tool that retrieves the operator schema that is most relevant to the prompt.
type OperatorsRAGTool struct {
	operators_db.OperatorsRetriever
}

func (ort *OperatorsRAGTool) Name() string {
	return "operatorsRAGTool"
}

func (ort *OperatorsRAGTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name: ort.Name(),
			Description: `Retrieve the operator information that is most relevant to the prompt.
						  This tool is used to retrieve kubernetes operators information before answering a relevant user prompt.
						  This tool should be used before generating answers from nothing. Do not over-use with the same prompt.`,

			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type": "string",
						"description": `The prompt to retrieve the operator schemas for.
										This prompt should be generalized and enhanced by you for best results.`,
					},
				},
				"required": []string{"prompt"},
			},
		},
	}
}

func (ort *OperatorsRAGTool) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
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

	schemas, err := ort.RetrieveOperators(ctx, args.Prompt)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to get relevant operators: %v", err),
		}
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    operators_db.OperatorSchemasToString(schemas),
	}
}
