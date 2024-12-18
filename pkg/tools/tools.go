package tools

import (
	"context"
	"github.com/tmc/langchaingo/llms"
)

// Tool abstracts the concept of a tool in the context of the usable LLM API.
type Tool interface {
	Name() string
	LLMTool() *llms.Tool
	Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse
}
