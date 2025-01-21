package steps

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// ToolStep implements Step for tool calling.
// It conveniently wraps llms.GenerateContent and its utility functions.
type ToolStep struct {
	call llms.FunctionCall
}

// NewToolStep creates a new tool step that serves as a proxy for invoking tool-calls.
// No model/human is involved in this step.
func NewToolStep(call llms.FunctionCall) *ToolStep {
	return &ToolStep{call: call}
}

// Execute runs the step with the given llm and returns the response.
func (t *ToolStep) Execute(_ context.Context) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: "KueryFlow tool call",
				ToolCalls: []llms.ToolCall{
					{ID: "kueryflow-tool",
						Type:         "function",
						FunctionCall: &t.call},
				},
			},
		}}, nil
}

// ToMessageContent converts a response to an AI message content.
func (t *ToolStep) ToMessageContent(response *llms.ContentResponse) llms.MessageContent {
	return llms.TextParts(llms.ChatMessageTypeTool, response.Choices[0].Content)
}

// WithHistory extends (to head) or replaces the history of the step.
// The function assumes that history will not be mutated after this call.
func (t *ToolStep) WithHistory(_ []llms.MessageContent, _ bool) Step {
	return t
}

// WithCallOptions extends (to tail) the call options of the step.
// The function assumes that call options will not be mutated after this call.
func (t *ToolStep) WithCallOptions(_ []llms.CallOption) Step {
	return t
}
