package flows

import (
	"context"
	"encoding/json"
	"fmt"
	clientset "github.com/kube-agent/kuery/pkg/generated/clientset/versioned"

	"github.com/tmc/langchaingo/llms"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/kube-agent/kuery/api/core/v1alpha1"
	"github.com/kube-agent/kuery/pkg/flows/steps"
)

// importKueryFlowTool is a tool that can get or execute a KueryFlow object.
// TODO: think of execution modes, add side-quests for recalcing missing step args.
type importKueryFlowTool struct {
	client clientset.Interface
	chain  Chain
	llm    llms.Model
}

func (t *importKueryFlowTool) Name() string {
	return "ImportKueryFlow"
}

func (t *importKueryFlowTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name: t.Name(),
			Description: `ImportKueryFlow is a tool that is used for getting or executing KueryFlows. These functionalities
							are split because in general you should not execute a KueryFlow without the user's consent.

							VERY IMPORTANT: a KueryFlow may include tool-calls with arguments that should be recalculated.
							Such steps may be preceded by 'ai' or 'human' steps to give you or the human the chance to provide or calculate data.`,

			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operation": map[string]interface{}{
						"type": "string",
						"description": `The operation to perform: GET, EXECUTE
										GET: Get a KueryFlow object by namespaced name.
										EXECUTE: Execute a KueryFlow object by namespaced name.`,
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the KueryFlow object to get or execute.",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "The namespace of the KueryFlow object to get or execute.",
					},
				},
				"required": []string{"operation", "name", "namespace"},
			},
		},
	}
}

type importCallArgs struct {
	Operation string `json:"operation"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (t *importKueryFlowTool) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args importCallArgs

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	kueryFlow, err := t.client.CoreV1alpha1().KueryFlows(args.Namespace).Get(ctx, args.Name, metav1.GetOptions{})
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to get KueryFlow: %v", err),
		}
	}

	switch args.Operation {
	case "GET":
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("%v", kueryFlow),
		}
	case "EXECUTE":
		if t.chain == nil || t.llm == nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    "no chain or llm set",
			}
		}

		if err := t.appendKueryFlowToChain(kueryFlow); err != nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    fmt.Sprintf("failed to append KueryFlow to chain: %v", err),
			}
		}

		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("executed KueryFlow: %v", kueryFlow.Name),
		}

	default:
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("unknown operation: %v", args.Operation),
		}
	}
}

func (t *importKueryFlowTool) appendKueryFlowToChain(kueryFlow *corev1alpha1.KueryFlow) error {
	// iterate in reverse order to append steps in the correct order
	for i := len(kueryFlow.Spec.Steps) - 1; i >= 0; i-- {
		step := kueryFlow.Spec.Steps[i]
		if err := t.validateStep(step); err != nil {
			return fmt.Errorf("failed to validate step: %v", err)
		}

		switch step.Type {
		case corev1alpha1.StepTypeTool:
			t.chain.PushNext(t.createToolStep(step), true)
		case corev1alpha1.StepTypeAI:
			t.chain.PushNext(steps.NewLLMStep(t.llm).WithHistory([]llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeAI, *step.AIPrompt),
			}, false), true)
		case corev1alpha1.StepTypeHuman:
			t.chain.PushNext(steps.NewHumanStep(steps.ReadFromSTDIN), true)
		default:
			return fmt.Errorf("unknown step type: %v", step.Type)
		}
	}

	return nil
}

const toolStepContext = `You are required to run the following tool-call (part of a KueryFlow), 
						but some of its arguments need to be figured out. Use the 'AddStep' tool in order to instruct your
						self further (e.g. to plan further, or to simply ask something from the user).`

func (t *importKueryFlowTool) validateStep(step corev1alpha1.Step) error {
	switch step.Type {
	case corev1alpha1.StepTypeTool:
		if step.FunctionCall == nil {
			return fmt.Errorf("missing function call in tool step")
		}
	case corev1alpha1.StepTypeAI:
		if step.AIPrompt == nil {
			return fmt.Errorf("missing AI prompt in AI step")
		}
	case corev1alpha1.StepTypeHuman:
	default:
		return fmt.Errorf("unknown step type: %v", step.Type)
	}

	return nil
}

func (t *importKueryFlowTool) createToolStep(step corev1alpha1.Step) steps.Step {
	if len(step.ArgsToRecalculate) > 0 {
		// in this case we need to add an instructional AI step to possibly start a chain of recalculations
		// to figure out the correct values for the arguments
		return steps.NewLLMStep(t.llm).WithHistory([]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeAI, toolStepContext)}, false)
	}

	// in this case we can simply create a tool step
	return steps.NewToolStep(*step.FunctionCall)
}
