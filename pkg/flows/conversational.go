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
	chain   Chain
	toolMgr *tools.Manager

	systemPrompt string
}

// NewConversationalFlow creates a new conversational flow.
func NewConversationalFlow(systemPrompt string, llm llms.Model, toolMgr *tools.Manager) *ConversationalFlow {
	return &ConversationalFlow{
		llm:          llm,
		chain:        NewChain(nil),
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

	f.chain.Reset()
	return f.execute(ctx, history)
}

// Loop executes the flow in a loop until the context is done.
func (f *ConversationalFlow) Loop(ctx context.Context) ([]llms.MessageContent, error) {
	history := make([]llms.MessageContent, 0)

	if f.systemPrompt != "" {
		history = appendHistory(ctx, history, llms.TextParts(llms.ChatMessageTypeSystem,
			[]string{f.systemPrompt}...))
	}

	for {
		select {
		case <-ctx.Done():
			return history, nil
		default:
			f.chain.Reset()
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
	logger := klog.FromContext(ctx)
	// iterate over chn.Next() until nil
	for {
		step := f.chain.Next()
		if step == nil {
			break
		}

		response, err := step.
			WithHistory(history, false).
			WithCallOptions([]llms.CallOption{llms.WithTools(f.getTools())}).
			Execute(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to execute step: %w", err)
		}

		history = appendHistory(ctx, history, step.ToMessageContent(response))
		// execute tool calls (if any) and add to history
		toolsUsed := false
		for _, msg := range f.toolMgr.ExecuteToolCalls(ctx, response) { // this could potentially add a step
			logger.V(4).Info("Tool Used", "content", msg.Parts)
			history = appendHistory(ctx, history, msg)
			toolsUsed = true
		}

		if toolsUsed { // add AI step to answer
			logger.V(4).Info("Added AI Step")
			f.chain.PushNext(steps.NewLLMStep(f.llm), true)
		}
	}

	return history, nil
}

func (f *ConversationalFlow) HumanStep(getter func(ctx context.Context) string) *ConversationalFlow {
	f.chain.Push([]steps.Step{
		steps.NewHumanStep(getter),
		steps.NewLLMStep(f.llm), // add AI step to answer
	})

	return f
}

func (f *ConversationalFlow) getTools() []llms.Tool {
	planner := plannerTool{
		chain: f.chain,
		llm:   f.llm,
	}

	return append(f.toolMgr.GetLLMTools(), *planner.LLMTool())
}

func appendHistory(ctx context.Context, history []llms.MessageContent, msg llms.MessageContent) []llms.MessageContent {
	logger := klog.FromContext(ctx)

	if msg.Role == "tool" {
		logger.V(3).Info("Tool", "content", msg.Parts)
	} else {
		logger.Info(string(msg.Role), "content", msg.Parts)
	}

	history = append(history, msg)
	return history
}
