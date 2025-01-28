package api

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/llms"
)

// ToolManager holds all available tools and streamlines operating them.
type ToolManager struct {
	tools map[string]Tool

	toolCallCache map[string]llms.ToolCall
	nextCallID    int

	// TODO: combine maps
	toolMaxRetries map[string]int
	toolApprovals  map[string]bool
	toolRetries    map[string]int // for starters, retries are global per LLM step
}

// NewToolManager creates a new ToolManager.
func NewToolManager() *ToolManager {
	return &ToolManager{
		tools:          make(map[string]Tool),
		toolCallCache:  make(map[string]llms.ToolCall),
		nextCallID:     1,
		toolMaxRetries: make(map[string]int),
		toolApprovals:  make(map[string]bool),
		toolRetries:    make(map[string]int),
	}
}

// WithTool adds a tool to the manager.
// The maxRetries parameter specifies the maximum number of consecutive runs a tool can have.
func (m *ToolManager) WithTool(tool Tool, maxRetries int) *ToolManager {
	m.tools[tool.Name()] = tool
	if tool.RequiresApproval() {
		m.toolApprovals[tool.Name()] = false
		m.toolRetries[tool.Name()] = 0
		m.toolMaxRetries[tool.Name()] = maxRetries
	}

	return m
}

// WithTools adds multiple tools to the manager.
// The maxRetries parameter specifies the maximum number of consecutive runs a tool can have.
// The tools and maxRetries slices must have the same length.
func (m *ToolManager) WithTools(tools []Tool, maxRetries []int) *ToolManager {
	if len(tools) != len(maxRetries) {
		return m
	}

	for idx, tool := range tools {
		m.WithTool(tool, maxRetries[idx])
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

	usedTools := []string{}

	for _, choice := range resp.Choices {
		for _, toolCall := range choice.ToolCalls {
			toolCallResponse, ok, requiresExplaining := m.callTool(ctx, &toolCall)
			newMessages = append(newMessages, llms.MessageContent{ // tool call message is always appended
				Role: llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{
					llms.TextPart(fmt.Sprintf("[ID: %d] Tool-Call %s executed\n",
						m.nextCallID, toolCall.FunctionCall.Name)),
					llms.ToolCall{
						ID:   toolCall.ID,
						Type: toolCall.Type,
						FunctionCall: &llms.FunctionCall{
							Name:      toolCall.FunctionCall.Name,
							Arguments: toolCall.FunctionCall.Arguments,
						},
					}}})

			requireFurtherProcessing = requireFurtherProcessing || requiresExplaining
			newMessages = append(newMessages, llms.MessageContent{
				Role:  llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{toolCallResponse}})

			if !ok {
				m.toolRetries[toolCall.FunctionCall.Name]++
				m.nextCallID++
				continue
			}

			// Bookkeeping, TODO: make this more maintainable
			m.toolCallCache[fmt.Sprintf("%d", m.nextCallID)] = toolCall // safe since execution blocks Kuery
			m.toolRetries[toolCall.FunctionCall.Name] = 0               // reset retries because tool was successful
			m.toolApprovals[toolCall.FunctionCall.Name] = false         // reset approval because tool was successful
			usedTools = append(usedTools, toolCall.FunctionCall.Name)
			m.nextCallID++
		}
	}

	return newMessages, requireFurtherProcessing
}

// getTool returns the tool with the given name.
func (m *ToolManager) getTool(name string) Tool {
	return m.tools[name]
}

// callTool calls the tool with the given tool call, if conditions allow.
// The returned tuple consists of:
// - The tool call response
// - A boolean indicating whether the tool call went through
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
		}, false, true
	}

	if m.toolRetries[tool.Name()] > m.toolMaxRetries[tool.Name()] {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    "tool has reached the maximum number of consecutive runs. Context should return to the user.",
		}, false, true
	} // this must be first to block AI retries in explanation windows

	if tool.RequiresApproval() && !m.toolApprovals[tool.Name()] {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    "tool requires explicit user approval before execution",
		}, false, true
	}

	response, ok := tool.Call(ctx, toolCall)
	return response, ok, tool.RequiresExplaining()
}

func (m *ToolManager) ApproveTools(toolNames []string) {
	for _, name := range toolNames {
		m.toolApprovals[name] = true
	}
}

func (m *ToolManager) ResetToolRetries() {
	for tool := range m.toolRetries {
		m.toolRetries[tool] = 0
	}
}
