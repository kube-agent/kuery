package tools

import "github.com/tmc/langchaingo/llms"

const (
	functionToolType = "function"
)

// Tool abstracts the concept of a tool in the context of the usable LLM API.
type Tool interface {
	Name() string
	LLMTool() *llms.Tool
	Call(toolCall *llms.ToolCall) llms.ToolCallResponse
}

// GetTools returns a list of all available tools.
func GetTools() []Tool {
	return []Tool{
		&WeatherTool{},
	}
}
