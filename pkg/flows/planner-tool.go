package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	"github.com/vMaroon/Kuery/pkg/flows"
	"github.com/vMaroon/Kuery/pkg/flows/steps"
)

// PlannerTool is a tool that can add a step of planning to a given chain.
type PlannerTool struct {
	Chain flows.Chain
	Model llms.Model
}

func (p *PlannerTool) Name() string {
	return "plannerTool"
}

func (p *PlannerTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name: "addPlanningStep",
			Description: `Extend the conversation flow with a planning step.
						  This tool should be used when a user request requires a multi-step plan for serving.
						  When planning is necessary, plan one step at a time and use this tool to add the step to the flow.`,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{
						"type":        "string",
						"description": "The instructional prompt for the planning step",
					},
				},
				"required": []string{"prompt"},
			},
		},
	}
}

func (p *PlannerTool) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
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

	if p.Chain == nil || p.Model == nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    "no chain or llm set",
		}
	}

	step := steps.NewLLMStep(p.Model).WithHistory([]llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeAI, args.Prompt),
	}, false)
	p.Chain.PushNext(step, true) // has to be true because the prompt would reset on chain::Reset

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    "Added planner step with prompt: " + args.Prompt,
	}
}
