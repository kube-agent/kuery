package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kube-agent/kuery/pkg/tools/api"

	"github.com/tmc/langchaingo/llms"

	operators_db "github.com/kube-agent/kuery/pkg/operators-db"
)

const functionToolType = "function"

var _ api.Tool = &OperatorsRAGTool{}

// OperatorsRAGTool is a tool that retrieves the operator schema that is most relevant to the prompt.
type OperatorsRAGTool struct {
	retriever operators_db.OperatorsRetriever
}

// NewOperatorsRAGTool creates a new OperatorsRAGTool.
func NewOperatorsRAGTool(retriever operators_db.OperatorsRetriever) *OperatorsRAGTool {
	return &OperatorsRAGTool{
		retriever: retriever,
	}
}

func (ort *OperatorsRAGTool) Name() string {
	return "operatorsRAGTool"
}

func (ort *OperatorsRAGTool) LLMTool() *llms.Tool {
	desc := `Retrieve the operator information that is most relevant to the prompt.
			This tool is used to retrieve kubernetes operators information before answering a relevant user prompt.
			This tool should be used before generating answers from nothing. Do not over-use with the same prompt.

			To install an operator, you must POST the Subscription specified in a schema to the dynamic K8s client.`

	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name:        ort.Name(),
			Description: api.AddApprovalRequirementToDescription(ort, desc),
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

func (ort *OperatorsRAGTool) Call(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool) {
	var args struct {
		Prompt string `json:"prompt"`
	}

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}, false
	}

	schemas, err := ort.retriever.RetrieveOperators(ctx, args.Prompt)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to get relevant operators: %v", err),
		}, false
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    operators_db.OperatorSchemasToString(schemas),
	}, true
}

// RequiresExplaining returns whether the tool requires explaining after
// execution.
func (ort *OperatorsRAGTool) RequiresExplaining() bool {
	return true
}

// RequiresApproval returns whether the tool requires approval before
// execution.
func (ort *OperatorsRAGTool) RequiresApproval() bool { return false }
