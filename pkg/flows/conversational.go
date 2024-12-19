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

	systemPrompt string
}

// NewConversationalFlow creates a new conversational flow.
func NewConversationalFlow(systemPrompt string, llm llms.Model, toolMgr *tools.Manager) *ConversationalFlow {
	return &ConversationalFlow{
		llm:          llm,
		toolMgr:      toolMgr,
		systemPrompt: systemPrompt,
	}
}

// Once executes the flow once.
func (f *ConversationalFlow) Once(ctx context.Context) ([]llms.MessageContent, error) {
	history := make([]llms.MessageContent, 0)

	if f.systemPrompt != "" {
		history = appendHistory(ctx, history, llms.TextParts(llms.ChatMessageTypeSystem,
			[]string{f.systemPrompt}...))
	}

	return f.execute(ctx, history)
}

// Loop executes the flow in a loop until the stop channel is closed.
func (f *ConversationalFlow) Loop(ctx context.Context, stopChan chan struct{}) ([]llms.MessageContent, error) {
	history := make([]llms.MessageContent, 0)

	if f.systemPrompt != "" {
		history = appendHistory(ctx, history, llms.TextParts(llms.ChatMessageTypeSystem,
			[]string{f.systemPrompt}...))
	}

	for {
		select {
		case <-stopChan:
			return history, nil
		default:
			executionHistory, err := f.execute(ctx, history)
			history = executionHistory
			if err != nil {
				return history, fmt.Errorf("failed to execute flow: %w", err)
			}
		}
	}
}

func (f *ConversationalFlow) execute(ctx context.Context,
	history []llms.MessageContent) ([]llms.MessageContent, error) {
	for i, step := range f.steps {
		response, err := step.
			WithHistory(history, true).
			WithCallOptions([]llms.CallOption{llms.WithTools(f.toolMgr.GetLLMTools())}).
			Execute(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to execute step %d: %w", i, err)
		}

		history = appendHistory(ctx, history, step.ToMessageContent(response))
		// execute tool calls (if any) and add to history
		toolsUsed := false
		for _, msg := range f.toolMgr.ExecuteToolCalls(ctx, response) {
			history = appendHistory(ctx, history, msg)
			toolsUsed = true
		}

		if toolsUsed { // add AI step to answer
			clarificationStep := steps.NewLLMStep(f.llm).WithHistory(history, true)
			response, err := clarificationStep.Execute(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to execute AI step after tool calls: %w", err)
			}

			history = appendHistory(ctx, history, clarificationStep.ToMessageContent(response))
		}
	}

	return history, nil
}

func (f *ConversationalFlow) HumanStep(getter func(ctx context.Context) string) *ConversationalFlow {
	f.steps = append(f.steps, steps.NewHumanStep(getter))
	// also append an AI step to answer
	f.steps = append(f.steps, steps.NewLLMStep(f.llm))

	return f
}

func appendHistory(ctx context.Context, history []llms.MessageContent, msg llms.MessageContent) []llms.MessageContent {
	logger := klog.FromContext(ctx)
	logger.Info("Step", "role", msg.Role, "content", msg.Parts)

	history = append(history, msg)
	return history
}
