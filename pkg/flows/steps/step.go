package steps

import (
	"context"
	"github.com/tmc/langchaingo/llms"
)

// Step abstracts a step in a flow.
type Step interface {
	// Execute runs a step with the given llm and returns the response.
	Execute(ctx context.Context) (*llms.ContentResponse, error)
	// ToMessageContent converts a response to an AI message content.
	ToMessageContent(response *llms.ContentResponse) llms.MessageContent
	// WithHistory extends (to head) or replaces the history of the step.
	WithHistory(history []llms.MessageContent, replace bool) Step
	// WithCallOptions extends (to tail) the call options of the step.
	WithCallOptions(callOptions []llms.CallOption) Step
}
