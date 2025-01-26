package kuery

import (
	"context"
	"fmt"
	"github.com/kube-agent/kuery/pkg/tools"

	"github.com/tmc/langchaingo/llms"

	"k8s.io/klog/v2"
)

// ToolManager holds all available tools and streamlines operating them.
type ToolManager struct {
	tools map[string]tools.Tool

	toolCallCache map[string]llms.ToolCall
	nextCallID    int
}

// NewToolManager creates a new ToolManager.
func NewToolManager() *ToolManager {
	return &ToolManager{
		tools:         make(map[string]tools.Tool),
		toolCallCache: make(map[string]llms.ToolCall),
		nextCallID:    1,
	}
}

// WithTool adds a tool to the manager.
func (m *ToolManager) WithTool(tool tools.Tool) *ToolManager {
	m.tools[tool.Name()] = tool
	return m
}

// WithTools adds multiple tools to the manager.
func (m *ToolManager) WithTools(tools []tools.Tool) *ToolManager {
	for _, tool := range tools {
		m.tools[tool.Name()] = tool
	}
	return m
}

// GetToolCall returns the tool call with the given ID.
func (m *ToolManager) GetToolCall(id string) (*llms.ToolCall, bool) {
	toolCall, ok := m.toolCallCache[id]
	return &toolCall, ok
}

// GetTool returns the tool with the given name.
func (m *ToolManager) GetTool(name string) tools.Tool {
	return m.tools[name]
}

// GetLLMTools returns all tools as LLM tools.
func (m *ToolManager) GetLLMTools() []llms.Tool {
	llmTools := make([]llms.Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		llmTools = append(llmTools, *tool.LLMTool())
	}

	return llmTools
}

// GetToolNames returns the names of all tools.
func (m *ToolManager) GetToolNames() []string {
	names := make([]string, 0, len(m.tools))
	for name := range m.tools {
		names = append(names, name)
	}

	return names
}

// ExecuteToolCalls executes the tool calls in the response and returns:
// - The new messages
// - A boolean indicating whether the response requires further processing (LLMStep).
//
// If the response does not contain any tool calls, it returns an empty slice with false.
func (m *ToolManager) ExecuteToolCalls(ctx context.Context, resp *llms.ContentResponse) ([]llms.MessageContent, bool) {
	newMessages := make([]llms.MessageContent, 0)
	logger := klog.FromContext(ctx)
	requireFurtherProcessing := false

	for _, choice := range resp.Choices {
		for _, toolCall := range choice.ToolCalls {
			tool := m.GetTool(toolCall.FunctionCall.Name)
			if tool == nil {
				logger.Info("tool not found", "toolCall", toolCall)
				continue
			}

			requireFurtherProcessing = requireFurtherProcessing || tool.RequiresExplaining()
			// append tool use
			m.toolCallCache[fmt.Sprintf("%d", m.nextCallID)] = toolCall // safe since execution blocks Kuery

			newMessages = append(newMessages, llms.MessageContent{
				Role: llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{
					llms.TextPart(fmt.Sprintf("Executing Tool-Call %s, ID: %d", tool.Name(), m.nextCallID)),
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
					tool.Call(ctx, &toolCall), // these count as in-parallel calls, therefore history is passed
					// without appending newMessages
				}})

			m.nextCallID++
		}
	}

	return newMessages, requireFurtherProcessing
}
