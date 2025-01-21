package tools

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// Tool abstracts the concept of a tool in the context of the usable LLM API.
type Tool interface {
	// Name returns the name of the tool.
	Name() string
	// LLMTool returns the tool as an LLM tool usable by the LLM API.
	LLMTool() *llms.Tool
	// Call executes the tool call and returns the response.
	Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse
}
