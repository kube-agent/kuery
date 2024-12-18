package flows

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	"github.com/vMaroon/Kuery/pkg/flows/steps"
	"github.com/vMaroon/Kuery/pkg/tools"
	"k8s.io/klog/v2"
)

// ConversationalFlow implements flow for pure-conversation flows.
type ConversationalFlow struct {
	llm     llms.Model
	steps   []steps.Step
	toolMgr *tools.Manager
}

// NewConversationalFlow creates a new conversational flow.
func NewConversationalFlow(llm llms.Model, toolMgr *tools.Manager) *ConversationalFlow {
	return &ConversationalFlow{
		llm:     llm,
		toolMgr: toolMgr,
	}
}

// Run executes the flow.
func (f *ConversationalFlow) Run(ctx context.Context) ([]llms.MessageContent, error) {
	logger := klog.FromContext(ctx)
	history := make([]llms.MessageContent, 0)

	for i, step := range f.steps {
		logger.V(3).Info("Running step", "index", i)

		response, err := step.
			WithHistory(history, true).
			WithCallOptions([]llms.CallOption{llms.WithTools(f.toolMgr.GetLLMTools())}).
			Execute(ctx)
		if err != nil {
			return history, fmt.Errorf("failed to execute step %d: %w", i, err)
		}

		logger.V(3).Info("Step executed", "index", i, "response", response)
		history = append(history, step.ToMessageContent(response))
		// execute tool calls (if any) and add to history
		toolsUsed := false
		for _, msg := range f.toolMgr.ExecuteToolCalls(ctx, response) {
			logger.V(3).Info("Tool call executed", "index", i, "response", msg)
			history = append(history, msg)
			toolsUsed = true
		}

		if toolsUsed { // add AI step to answer
			clarificationStep := steps.NewLLMStep(f.llm).WithHistory(history, true)
			response, err := clarificationStep.Execute(ctx)
			if err != nil {
				return history, fmt.Errorf("failed to execute AI step after tool calls: %w", err)
			}

			history = append(history, clarificationStep.ToMessageContent(response))
		}
	}

	return history, nil
}

// Append adds a step to the flow.
func (f *ConversationalFlow) Append(step steps.Step) *ConversationalFlow {
	f.steps = append(f.steps, step)
	return f
}

func (f *ConversationalFlow) HumanStep(prompt string) *ConversationalFlow {
	f.steps = append(f.steps, steps.NewHumanStep(func(ctx context.Context) string {
		return prompt
	}))

	// also append an AI step to answer
	f.steps = append(f.steps, steps.NewLLMStep(f.llm))

	return f
}
