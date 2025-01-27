package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kube-agent/kuery/pkg/flows"

	"github.com/tmc/langchaingo/llms"

	"github.com/kube-agent/kuery/pkg/flows/steps"
)

var _ Tool = &AddStepTool{}

// AddStepTool is a tool that can add a step of planning to a given chain.
type AddStepTool struct {
	chain flows.Chain
	llm   llms.Model
}

// NewAddStepTool creates a new AddStepTool.
func NewAddStepTool(chain flows.Chain, llm llms.Model) *AddStepTool {
	return &AddStepTool{
		chain: chain,
		llm:   llm,
	}
}

func (t *AddStepTool) Name() string {
	return "AddStep"
}

func (t *AddStepTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name: t.Name(),
			Description: `Extend the execution flow with a step.
						  This tool should be used when a user request requires a multi-step plan for serving.
						  When planning is necessary, plan one step at a time and use this tool to add the step to the flow.

						  This tool is useful for when you need to run again before giving the turn back to the user.`,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{
						"type":        "string",
						"description": "The instructional prompt for the added step, to be read by you in the next iteration.",
					},
				},
				"required": []string{"prompt"},
			},
		},
	}
}

func (t *AddStepTool) Call(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool) {
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

	if t.chain == nil || t.llm == nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    "no chain or llm set",
		}, false
	}

	step := steps.NewLLMStep(t.llm).WithHistory([]llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeAI, args.Prompt),
	}, false)
	t.chain.PushNext(step, true) // has to be true because the prompt would reset on chain::Reset

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    "Added AI step with prompt: " + args.Prompt,
	}, true
}

// RequiresExplaining returns whether the tool requires explaining after
// execution.
func (t *AddStepTool) RequiresExplaining() bool {
	return false // adds a step to the chain, no need to explain
}

// RequiresApproval returns whether the tool requires approval before
// execution.
func (t *AddStepTool) RequiresApproval() bool { return false }
