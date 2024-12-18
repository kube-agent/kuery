package steps

import (
	"context"
	"github.com/tmc/langchaingo/llms"
)

// HumanStep is a step that represents a human step in a flow.
type HumanStep struct {
	getter func(ctx context.Context) string
}

// NewHumanStep creates a new human step.
func NewHumanStep(getter func(ctx context.Context) string) *HumanStep {
	return &HumanStep{getter: getter}
}

// Execute runs a human step with the given llm and returns the response.
func (s *HumanStep) Execute(ctx context.Context) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: s.getter(ctx)},
		},
	}, nil
}

// ToMessageContent converts a response to an AI message content.
func (s *HumanStep) ToMessageContent(response *llms.ContentResponse) llms.MessageContent {
	return llms.TextParts(llms.ChatMessageTypeHuman, response.Choices[0].Content)
}

// WithGetter sets the getter of the step.
// The function assumes that getter is immutable.
func (s *HumanStep) WithGetter(getter func(ctx context.Context) string) *HumanStep {
	s.getter = getter
	return s
}

// WithHistory extends (to head) or replaces the history of the step.
// The function assumes that history will not be mutated after this call.
func (s *HumanStep) WithHistory(history []llms.MessageContent, replace bool) Step {
	// Human steps do not have history.
	return s
}

// WithCallOptions extends (to tail) the call options of the step.
// The function assumes that call options will not be mutated after this call.
func (s *HumanStep) WithCallOptions(callOptions []llms.CallOption) Step {
	// Human steps do not have call options.
	return s
}
