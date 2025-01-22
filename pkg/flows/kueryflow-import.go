package flows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kr/pretty"
	"github.com/tmc/langchaingo/llms"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/kube-agent/kuery/api/core/v1alpha1"
	"github.com/kube-agent/kuery/pkg/flows/steps"
	clientset "github.com/kube-agent/kuery/pkg/generated/clientset/versioned"
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
							are split because in general you should not execute a KueryFlow without the user's consent.`,

			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operation": map[string]interface{}{
						"type": "string",
						"description": `The operation to perform: LIST, GET, EXECUTE
										LIST: Get KueryFlow objects in a namespace.
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

	switch args.Operation {
	case "LIST":
		kueryFlows, err := t.listKueryFlows(ctx, args.Namespace)
		if err != nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    fmt.Sprintf("failed to list KueryFlows: %v", err),
			}
		}

		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("KueryFlows in namespace %v: %v", args.Namespace, kueryFlows),
		}
	case "GET":
		kueryFlow, err := t.getKueryFlow(ctx, args.Namespace, args.Name)
		if err != nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    fmt.Sprintf("failed to get KueryFlow: %v", err),
			}
		}

		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("KueryFlow: %v", kueryFlow),
		}
	case "EXECUTE":
		if t.chain == nil || t.llm == nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    "no chain or llm set",
			}
		}

		kueryFlow, err := t.getKueryFlow(ctx, args.Namespace, args.Name)
		if err != nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    fmt.Sprintf("failed to get KueryFlow: %v", err),
			}
		}

		t.appendKueryFlowToChain(kueryFlow)
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

func (t *importKueryFlowTool) listKueryFlows(ctx context.Context, namespace string) ([]corev1alpha1.KueryFlow, error) {
	kueryFlows, err := t.client.CoreV1alpha1().KueryFlows(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list KueryFlows: %v", err)
	}

	return kueryFlows.Items, nil
}

func (t *importKueryFlowTool) getKueryFlow(ctx context.Context, namespace, name string) (*corev1alpha1.KueryFlow, error) {
	kueryFlow, err := t.client.CoreV1alpha1().KueryFlows(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get KueryFlow: %v", err)
	}

	return kueryFlow, nil
}

func (t *importKueryFlowTool) appendKueryFlowToChain(kueryFlow *corev1alpha1.KueryFlow) {
	// iterate in reverse order to append steps in the correct order
	for i := len(kueryFlow.Spec.Steps) - 1; i >= 0; i-- {
		step := kueryFlow.Spec.Steps[i]
		t.chain.PushNext(t.createToolStep(step), true)
	}
}

const toolStepContext = `You are required to run the following tool-call (part of a KueryFlow), 
						but some of its arguments need to be figured out first - those listed in argsToRecalculate. 
						Use the 'AddStep' tool in order to instruct your self further to figure out the correct values.
						If required, you may ask the user to help you figure them out.`

func (t *importKueryFlowTool) createToolStep(step corev1alpha1.Step) steps.Step {
	if len(step.ArgsToRecalculate) > 0 {
		// in this case we need to add an instructional AI step to possibly start a chain of recalculations
		// to figure out the correct values for the arguments
		return steps.NewLLMStep(t.llm).WithHistory([]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeAI, pretty.Sprint("%s:\n%v", toolStepContext, *step.FunctionCall))}, false)
	}

	// in this case we can simply create a tool step
	return steps.NewLLMStep(t.llm).WithHistory([]llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeAI, pretty.Sprint("Execute the following tool-call:\n%v",
			*step.FunctionCall))}, false)

}
