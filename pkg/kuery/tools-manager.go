package kuery

import (
	"context"
	"fmt"
	"github.com/kube-agent/kuery/pkg/tools"

	"github.com/tmc/langchaingo/llms"
)

// ToolManager holds all available tools and streamlines operating them.
type ToolManager struct {
	maxRetries int
	tools      map[string]tools.Tool

	toolCallCache map[string]llms.ToolCall
	nextCallID    int

	toolApprovals map[string]bool
}

// NewToolManager creates a new ToolManager.
func NewToolManager(maxRetries int) *ToolManager {
	return &ToolManager{
		maxRetries:    maxRetries,
		tools:         make(map[string]tools.Tool),
		toolCallCache: make(map[string]llms.ToolCall),
		nextCallID:    1,
		toolApprovals: make(map[string]bool),
	}
}

// WithTool adds a tool to the manager.
func (m *ToolManager) WithTool(tool tools.Tool) *ToolManager {
	m.tools[tool.Name()] = tool
	if tool.RequiresApproval() {
		m.toolApprovals[tool.Name()] = false
	}

	return m
}

// WithTools adds multiple tools to the manager.
func (m *ToolManager) WithTools(tools []tools.Tool) *ToolManager {
	for _, tool := range tools {
		m.WithTool(tool)
	}
	return m
}

// GetToolCall returns the tool call with the given ID.
func (m *ToolManager) GetToolCall(id string) (*llms.ToolCall, bool) {
	toolCall, ok := m.toolCallCache[id]
	return &toolCall, ok
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
	requireFurtherProcessing := false

	for _, choice := range resp.Choices {
		for _, toolCall := range choice.ToolCalls {
			newMessages = append(newMessages, llms.MessageContent{ // tool call message is always appended
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

			toolCallResponse, blocked, requiresExplaining := m.callTool(ctx, &toolCall)
			if blocked {
				newMessages = append(newMessages, llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.TextPart(fmt.Sprintf("Tool-Call %s blocked: %s\n", toolCall.FunctionCall.Name,
							toolCallResponse.Content)),
					}})
				continue
			}

			requireFurtherProcessing = requireFurtherProcessing || requiresExplaining

			// append tool response
			newMessages = append(newMessages, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.TextPart(fmt.Sprintf("[ID: %d] Tool-Call %s executed\n",
						m.nextCallID, toolCall.FunctionCall.Name)),
					toolCallResponse}})

			// save tool use
			m.toolCallCache[fmt.Sprintf("%d", m.nextCallID)] = toolCall // safe since execution blocks Kuery
			m.nextCallID++
		}
	}

	return newMessages, requireFurtherProcessing
}

// getTool returns the tool with the given name.
func (m *ToolManager) getTool(name string) tools.Tool {
	return m.tools[name]
}

// callTool calls the tool with the given tool call, if conditions allow.
// The returned tuple consists of:
// - The tool call response
// - A boolean indicating whether the tool call was successful or blocked
// - A boolean indicating whether the tool call requires explaining
//
// If a toolcall is blocked, the response would contain the reason.
func (m *ToolManager) callTool(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool, bool) {
	tool := m.getTool(toolCall.FunctionCall.Name)
	if tool == nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("tool not found: %s", toolCall.FunctionCall.Name),
		}, false, false
	}

	if tool.GetFailedCallCount(toolCall) >= m.maxRetries {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    "tool has reached the maximum number of consecutive runs. Context should return to the user.",
		}, false, false
	} // this must be first to block AI retries in explanation windows

	if tool.RequiresApproval() && !m.toolApprovals[tool.Name()] {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    "tool requires explicit user approval before execution",
		}, false, tool.RequiresExplaining()
	}

	response, ok := tool.Call(ctx, toolCall)
	return response, ok, tool.RequiresExplaining()
}
