package kuery

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/kr/pretty"
	"github.com/kube-agent/kuery/pkg/flows"
	"github.com/kube-agent/kuery/pkg/tools/api"
	"github.com/tmc/langchaingo/llms"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/kube-agent/kuery/pkg/flows/steps"
	clientset "github.com/kube-agent/kuery/pkg/generated/clientset/versioned"
	"github.com/kube-agent/kuery/pkg/tools"
)

// ConversationalFlow implements flow for pure-conversation flows.
type ConversationalFlow struct {
	llm     llms.Model
	chain   flows.Chain
	toolMgr *api.ToolManager

	systemPrompt string
}

// NewConversationalFlow creates a new conversational flow.
func NewConversationalFlow(systemPrompt string, llm llms.Model, toolMgr *api.ToolManager,
	cfg *rest.Config) *ConversationalFlow {
	chain := flows.NewChain(nil)

	planner := tools.NewAddStepTool(chain, llm)
	toolMgr = toolMgr.WithTool(planner, 1)

	if cfg != nil {
		coreClient, err := clientset.NewForConfig(cfg)
		if err == nil {
			importKueryFlowTool := tools.NewImportKueryFlowTool(coreClient, chain, toolMgr, llm)

			exportKueryFlowTool := tools.NewExportKueryFlowTool(coreClient, toolMgr.GetToolCall)

			toolMgr = toolMgr.WithTool(importKueryFlowTool, 3).WithTool(exportKueryFlowTool, 3)
		} else {
			klog.Error("failed to create core client", "error", err)
		}
	}

	toolsApprovalTool := tools.NewToolApprovalTool(chain, llm, toolMgr)
	toolMgr = toolMgr.WithTool(toolsApprovalTool, 2)

	return &ConversationalFlow{
		llm:          llm,
		chain:        chain,
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
	logger := klog.FromContext(ctx)

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
				logger.Error(err, "failed to execute flow")
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

		if step.Type() == steps.StepTypeHuman {
			f.toolMgr.ResetToolRetries() // done here to avoid the LLM rampage with tool calls in consecutive turns
		}

		response, err := step.
			WithHistory(history, true).
			WithCallOptions([]llms.CallOption{llms.WithTools(f.toolMgr.GetLLMTools())}).
			Execute(ctx)
		if err != nil {
			return history, fmt.Errorf("failed to execute step: %w", err)
		}

		history = appendHistory(ctx, history, step.ToMessageContent(response))

		msgs, requiresClarificationStep := f.toolMgr.ExecuteToolCalls(ctx, response)
		for _, msg := range msgs { // this could potentially add a step
			logger.V(4).Info("Tool Used", "content", msg.Parts)
			history = appendHistory(ctx, history, msg)
		}

		if requiresClarificationStep {
			logger.V(2).Info("Added AI Step")
			f.chain.PushNext(steps.NewLLMStep(f.llm), true)
		}
	}

	return history, nil
}

// HumanStep appends a human-driven step to the flow. The addition of the step
// will be followed by an AI step to answer.
func (f *ConversationalFlow) HumanStep(getter func(ctx context.Context) string) *ConversationalFlow {
	f.chain.Push([]steps.Step{
		steps.NewHumanStep(getter),
		steps.NewLLMStep(f.llm), // add AI step to answer
	})

	return f
}

func appendHistory(ctx context.Context, history []llms.MessageContent, msg llms.MessageContent) []llms.MessageContent {
	c := color.New(color.FgHiWhite)
	switch msg.Role {
	case llms.ChatMessageTypeSystem:
		fallthrough
	case llms.ChatMessageTypeAI:
		c = color.New(color.FgCyan)
	case llms.ChatMessageTypeHuman:
		c = color.New(color.FgHiBlue)
	case llms.ChatMessageTypeTool:
		c = color.New(color.FgHiGreen)
	}

	// if tool, trim parts
	if msg.Role == llms.ChatMessageTypeTool {
		c.Println(slicedString(pretty.Sprintf("[%s]: %s", msg.Role, msg.Parts), 200))
	} else {
		c.Println(pretty.Sprintf("[%s]: %s", msg.Role, msg.Parts))
	}

	history = append(history, msg)
	return history
}

func slicedString(str string, length int) string {
	if len(str) > length {
		return str[:length] + "..."
	}
	return str
}
