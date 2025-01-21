package steps

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// LLMStep implements Step for LLM models.
// It conveniently wraps llms.GenerateContent and its utility functions.
type LLMStep struct {
	model       llms.Model
	history     []llms.MessageContent
	callOptions []llms.CallOption
}

// NewLLMStep creates a new LLM step.
func NewLLMStep(model llms.Model) *LLMStep {
	return &LLMStep{model: model}
}

// Execute runs the step with the given llm and returns the response.
func (s *LLMStep) Execute(ctx context.Context) (*llms.ContentResponse, error) {
	return s.model.GenerateContent(ctx, s.history, s.callOptions...)
}

// ToMessageContent converts a response to an AI message content.
func (s *LLMStep) ToMessageContent(response *llms.ContentResponse) llms.MessageContent {
	return llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)
}

// WithModel sets the model of the step.
// The function assumes that model will remain available after this call.
func (s *LLMStep) WithModel(model llms.Model) *LLMStep {
	s.model = model
	return s
}

// WithHistory extends (to head) or replaces the history of the step.
// The function assumes that history will not be mutated after this call.
func (s *LLMStep) WithHistory(history []llms.MessageContent, replace bool) Step {
	if replace {
		s.history = history
	} else {
		s.history = append(history, s.history...)
	}

	return s
}

// WithCallOptions extends (to tail) the call options of the step.
// The function assumes that call options will not be mutated after this call.
func (s *LLMStep) WithCallOptions(callOptions []llms.CallOption) Step {
	s.callOptions = append(s.callOptions, callOptions...)
	return s
}

// WithPrompt adds a prompt to the end of the history.
func (s *LLMStep) WithPrompt(prompt string) *LLMStep {
	s.history = append(s.history, llms.TextParts(llms.ChatMessageTypeHuman, prompt))
	return s
}
