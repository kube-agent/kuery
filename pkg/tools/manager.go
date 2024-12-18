package tools

import (
	"context"
	"github.com/tmc/langchaingo/llms"
)

// Manager is a tool manager that holds all available tools, and provides methods
// to work with them.
type Manager struct {
	tools map[string]Tool
}

// NewManager creates a new tool manager.
func NewManager() *Manager {
	return &Manager{
		tools: make(map[string]Tool),
	}
}

// WithTool adds a tool to the manager.
func (m *Manager) WithTool(tool Tool) *Manager {
	m.tools[tool.Name()] = tool
	return m
}

// WithTools adds multiple tools to the manager.
func (m *Manager) WithTools(tools ...Tool) *Manager {
	for _, tool := range tools {
		m.tools[tool.Name()] = tool
	}
	return m
}

// GetTool returns the tool with the given name.
func (m *Manager) GetTool(name string) Tool {
	return m.tools[name]
}

// GetLLMTools returns all tools as LLM tools.
func (m *Manager) GetLLMTools() []llms.Tool {
	llmTools := make([]llms.Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		llmTools = append(llmTools, *tool.LLMTool())
	}

	return llmTools
}

// ExecuteToolCalls executes the tool calls in the response and returns the new messages.
// If the response does not contain any tool calls, it returns an empty slice.
func (m *Manager) ExecuteToolCalls(ctx context.Context, resp *llms.ContentResponse) []llms.MessageContent {
	newMessages := make([]llms.MessageContent, 0)
	for _, choice := range resp.Choices {
		for _, toolCall := range choice.ToolCalls {
			tool := m.GetTool(toolCall.FunctionCall.Name)
			if tool == nil {
				continue
			}

			// append tool use
			newMessages = append(newMessages, llms.MessageContent{
				Role: llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{
					llms.ToolCall{
						ID:   toolCall.ID,
						Type: toolCall.Type,
						FunctionCall: &llms.FunctionCall{
							Name:      toolCall.FunctionCall.Name,
							Arguments: toolCall.FunctionCall.Arguments,
						},
					}}})
			// append tool response
			newMessages = append(newMessages, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					tool.Call(ctx, &toolCall),
				}})
		}
	}

	return newMessages
}
