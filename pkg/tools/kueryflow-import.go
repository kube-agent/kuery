package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kube-agent/kuery/pkg/flows"
	"github.com/kube-agent/kuery/pkg/tools/api"

	"github.com/tmc/langchaingo/llms"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/kube-agent/kuery/api/core/v1alpha1"
	"github.com/kube-agent/kuery/pkg/flows/steps"
	clientset "github.com/kube-agent/kuery/pkg/generated/clientset/versioned"
)

// ImportKueryFlowTool is a tool that can get or execute a KueryFlow object.
// TODO: Add side-quests for recalculating missing step args.
// TODO: Right now the model gets "execute X call" for simplicity, in the future the model should not be involved
type ImportKueryFlowTool struct {
	client  clientset.Interface
	chain   flows.Chain
	toolMgr *api.ToolManager
	llm     llms.Model
}

// NewImportKueryFlowTool creates a new ImportKueryFlowTool.
func NewImportKueryFlowTool(client clientset.Interface, chain flows.Chain,
	toolMgr *api.ToolManager, llm llms.Model) *ImportKueryFlowTool {
	return &ImportKueryFlowTool{
		client:  client,
		chain:   chain,
		toolMgr: toolMgr,
		llm:     llm,
	}
}

func (t *ImportKueryFlowTool) Name() string {
	return "ImportKueryFlow"
}

func (t *ImportKueryFlowTool) LLMTool() *llms.Tool {
	desc := `ImportKueryFlow is a tool that is used for getting or executing KueryFlows. These functionalities
			are split because in general you should not execute a KueryFlow without the user's consent.`

	return &llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        t.Name(),
			Description: api.AddApprovalRequirementToDescription(t, desc),
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

func (t *ImportKueryFlowTool) Call(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool) {
	var args importCallArgs

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}, false
	}

	switch args.Operation {
	case "LIST":
		kueryFlows, err := t.listKueryFlows(ctx, args.Namespace)
		if err != nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    fmt.Sprintf("failed to list KueryFlows: %v", err),
			}, false
		}

		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("KueryFlows in namespace %v: %v", args.Namespace, kueryFlows),
		}, true
	case "GET":
		kueryFlow, err := t.getKueryFlow(ctx, args.Namespace, args.Name)
		if err != nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    fmt.Sprintf("failed to get KueryFlow: %v", err),
			}, false
		}

		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("KueryFlow: %v", kueryFlow),
		}, true
	case "EXECUTE":
		if t.chain == nil || t.llm == nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    "no chain or llm set",
			}, false
		}

		kueryFlow, err := t.getKueryFlow(ctx, args.Namespace, args.Name)
		if err != nil {
			return llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    fmt.Sprintf("failed to get KueryFlow: %v", err),
			}, false
		}

		t.appendKueryFlowToChain(kueryFlow)
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("loaded KueryFlow steps: %v", kueryFlow.Name),
		}, true

	default:
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("unknown operation: %v", args.Operation),
		}, false
	}
}

// RequiresExplaining returns whether the tool requires explaining after
// execution.
func (t *ImportKueryFlowTool) RequiresExplaining() bool {
	return false
}

// RequiresApproval returns whether the tool requires approval before
// execution.
func (t *ImportKueryFlowTool) RequiresApproval() bool { return true }

func (t *ImportKueryFlowTool) listKueryFlows(ctx context.Context, namespace string) ([]corev1alpha1.KueryFlow, error) {
	kueryFlows, err := t.client.CoreV1alpha1().KueryFlows(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list KueryFlows: %v", err)
	}

	return kueryFlows.Items, nil
}

func (t *ImportKueryFlowTool) getKueryFlow(ctx context.Context, namespace, name string) (*corev1alpha1.KueryFlow, error) {
	kueryFlow, err := t.client.CoreV1alpha1().KueryFlows(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get KueryFlow: %v", err)
	}

	return kueryFlow, nil
}

func (t *ImportKueryFlowTool) appendKueryFlowToChain(kueryFlow *corev1alpha1.KueryFlow) {
	// iterate in reverse order to append steps in the correct order
	for i := len(kueryFlow.Spec.Steps) - 1; i >= 0; i-- {
		step := kueryFlow.Spec.Steps[i]
		t.chain.PushNext(steps.NewLLMStep(t.llm), true) // LLM step to handle the tool step
		t.chain.PushNext(t.createToolStep(step), true)  // this will execute before the above
	}
}

const toolStepContext = `You are required to run the following tool-call (part of a KueryFlow), 
						but some of its arguments need to be figured out first - those listed in argsToRecalculate. 
						Use the 'AddStep' tool in order to instruct your self further to figure out the correct values.
						If required, you may ask the user to help you figure them out.`

func (t *ImportKueryFlowTool) createToolStep(step corev1alpha1.Step) steps.Step {
	if len(step.ArgsToRecalculate) > 0 {
		// in this case we need to add an instructional AI step to possibly start a chain of recalculations
		// to figure out the correct values for the arguments
		return steps.NewHumanStep(func(_ context.Context) string {
			return fmt.Sprintf("%s:\n%v", toolStepContext, *step.FunctionCall)
		})
	}

	t.toolMgr.ApproveTools([]string{step.FunctionCall.Name}) // approve the tool to be executed
	// in this case we can simply create a tool step
	return steps.NewHumanStep(func(_ context.Context) string {
		return fmt.Sprintf("Execute the following tool-call:\n%v", *step.FunctionCall)
	})
}
