package api

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

// Tool abstracts the concept of a tool in the context of the usable LLM API.
type Tool interface {
	// Name returns the name of the tool.
	Name() string
	// LLMTool returns the tool as an LLM tool usable by the LLM API.
	LLMTool() *llms.Tool
	// Call executes the tool call and returns the response and whether the
	// call was successful.
	Call(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool)
	// RequiresExplaining returns whether the tool requires explaining after
	// execution.
	RequiresExplaining() bool
	// RequiresApproval returns whether the tool requires approval before
	// execution.
	RequiresApproval() bool
}

func AddApprovalRequirementToDescription(tool Tool, description string) string {
	if tool.RequiresApproval() {
		return fmt.Sprintf("%s\nIMPORTANT: THIS TOOL REQUIRES EXPLICIT USER CONSENT, "+
			"USE 'RequestApprovalForTools' TOOL FIRST.", description)
	}
	return description
}
