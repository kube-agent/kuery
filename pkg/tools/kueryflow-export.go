package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kube-agent/kuery/pkg/tools/api"

	"github.com/tmc/langchaingo/llms"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/kube-agent/kuery/api/core/v1alpha1"
	clientset "github.com/kube-agent/kuery/pkg/generated/clientset/versioned"
)

var _ api.Tool = &ExportKueryFlowTool{}

// ExportKueryFlowTool is a tool that can can export a KueryFlow from a
// conversation.
type ExportKueryFlowTool struct {
	client clientset.Interface
	// toolCallGetter is a function that can get a tool-call by ID.
	toolCallGetter func(string) (*llms.ToolCall, bool)
}

// NewExportKueryFlowTool creates a new ExportKueryFlowTool.
func NewExportKueryFlowTool(client clientset.Interface,
	toolCallGetter func(string) (*llms.ToolCall, bool)) *ExportKueryFlowTool {
	return &ExportKueryFlowTool{
		client:         client,
		toolCallGetter: toolCallGetter,
	}
}

func (t *ExportKueryFlowTool) Name() string {
	return "exportKueryFlow"
}

func (t *ExportKueryFlowTool) LLMTool() *llms.Tool {
	desc := `ExportKueryFlow is a tool that is used to export a KueryFlow from the active conversation.
			A KueryFlow is a CR that contains a sequence of tool-calls. An exported KueryFlow can be
			later executed using the ImportKueryFlow tool.

			A flow of tool-calls may be completely deterministic if it contains concrete argument values,
			or indeterministic if it contains arguments that should be recalculated upon execution.

			YOU SHOULD ALWAYS ALWAYS PREFER deterministic tool-calls when possible.
			The user should be fully aware of what you're exporting before it is done.`

	return &llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        t.Name(),
			Description: api.AddApprovalRequirementToDescription(t, desc),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
						"description": `The name of the KueryFlow object to be exported.
										Prefer short, lowercase and precise names, recall kubernetes naming conventions`,
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "The namespace of the KueryFlow object to be exported.",
					},
					"steps": map[string]interface{}{
						"type":        "array",
						"description": "The steps in the flow.",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"toolCallID": map[string]interface{}{
									"type": "string",
									"description": `The ID of the tool-call in the history.
													Typically a tool-call is prefixed with: "Executing Tool-Call <name>, ID: <id>"`,
								},
								"argsToRecalculate": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "string",
									},
									"description": `A list of the names of arguments that should be recalculated upon execution.`,
								},
							},
							"required": []string{"toolCallID"},
						},
					},
				},
				"required": []string{"name", "namespace", "steps"},
			},
		},
	}
}

type toolCallRef struct {
	ID                string   `json:"toolCallID"`
	ArgsToRecalculate []string `json:"argsToRecalculate"`
}

type exportCallArgs struct {
	Name      string        `json:"name"`
	Namespace string        `json:"namespace"`
	Steps     []toolCallRef `json:"steps"`
}

func (t *ExportKueryFlowTool) Call(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool) {
	var args exportCallArgs

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}, false
	}

	if err := t.createOrUpdateKueryFlow(ctx, &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to export KueryFlow: %v", err),
		}, false
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    "Exported KueryFlow: " + args.Name,
	}, true
}

// RequiresExplaining returns whether the tool requires explaining after
// execution.
func (t *ExportKueryFlowTool) RequiresExplaining() bool {
	return true
}

// RequiresApproval returns whether the tool requires approval before
// execution.
func (t *ExportKueryFlowTool) RequiresApproval() bool { return true }
func (t *ExportKueryFlowTool) createOrUpdateKueryFlow(ctx context.Context, args *exportCallArgs) error {
	var kfSteps []corev1alpha1.Step

	for _, step := range args.Steps {
		call, ok := t.toolCallGetter(step.ID)
		if !ok {
			return fmt.Errorf("tool call not found: %v", step.ID)
		}

		kfSteps = append(kfSteps, corev1alpha1.Step{
			FunctionCall:      call.FunctionCall,
			ArgsToRecalculate: step.ArgsToRecalculate,
		})
	}

	kueryFlow := &corev1alpha1.KueryFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      args.Name,
			Namespace: args.Namespace,
		},
		Spec: corev1alpha1.KueryFlowSpec{
			Steps: kfSteps,
		},
	}

	_, err := t.client.CoreV1alpha1().KueryFlows(args.Namespace).Create(ctx, kueryFlow, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create KueryFlow: %v", err)
		}

		_, err = t.client.CoreV1alpha1().KueryFlows(args.Namespace).Update(ctx, kueryFlow, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update KueryFlow: %v", err)
		}
	}

	return nil
}
