package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kube-agent/kuery/pkg/flows"
	"github.com/kube-agent/kuery/pkg/tools/api"
	"strings"

	"github.com/tmc/langchaingo/llms"

	"github.com/kube-agent/kuery/pkg/flows/steps"
)

var _ api.Tool = &ToolApprovalTool{}

// ToolApprovalTool is a tool that the LLM can use to request for explicit approval on a tool-use.
type ToolApprovalTool struct {
	chain   flows.Chain
	llm     llms.Model
	toolMgr *api.ToolManager
}

// NewToolApprovalTool creates a new ToolApprovalTool.
func NewToolApprovalTool(chain flows.Chain, llm llms.Model, toolMgr *api.ToolManager) *ToolApprovalTool {
	return &ToolApprovalTool{
		chain:   chain,
		llm:     llm,
		toolMgr: toolMgr,
	}
}

func (t *ToolApprovalTool) Name() string {
	return "RequestApprovalForTools"
}

func (t *ToolApprovalTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name: t.Name(),
			Description: `RequestApprovalForTool is a tool that is used to request for explicit approval before
						  executing tools that require approval.
							An approval is valid until successful execution.`,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"toolNames": map[string]any{
						"type":        "array",
						"description": "The names of the tools to request approval for.",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required": []string{"toolNames"},
			},
		},
	}
}

const toolApprovalText = `The human will be prompted for approval after your next turn.
						  Explain to the user that they should explicitly write 'yes' to approve the tools to be ran.
							Be transparent and let them know exactly what they're approving.`

func (t *ToolApprovalTool) Call(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool) {
	var args struct {
		ToolNames []string `json:"toolNames"`
	}

	if t.toolMgr == nil || t.llm == nil || t.chain == nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    t.Name() + " is not properly initialized",
		}, false
	}

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}, false
	}

	// push a human step that sets the approval flag FOLLOWED by an AI step to continue the flow
	step := steps.NewHumanStep(func(ctx context.Context) string {
		humanInput := steps.ReadFromSTDIN(ctx)
		if strings.ToLower(humanInput) == "yes" {
			t.toolMgr.ApproveTools(args.ToolNames)
		}

		return humanInput
	})

	t.chain.PushNext(steps.NewLLMStep(t.llm), true) // reverse order cuz PushNext
	t.chain.PushNext(step, true)

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    toolApprovalText,
	}, true
}

// RequiresExplaining returns whether the tool requires explaining after
// execution.
func (t *ToolApprovalTool) RequiresExplaining() bool {
	return true // AI should explain the approval request
}

// RequiresApproval returns whether the tool requires approval before
// execution.
func (t *ToolApprovalTool) RequiresApproval() bool { return false }
