package flows

import (
	"context"
	"encoding/json"
	"fmt"
	corev1alpha1 "github.com/kube-agent/kuery/api/core/v1alpha1"
	clientset "github.com/kube-agent/kuery/pkg/generated/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tmc/langchaingo/llms"

	"github.com/kube-agent/kuery/pkg/tools"
)

var _ tools.Tool = &exportKueryFlowTool{}

// exportKueryFlowTool is a tool that can can export a KueryFlow from a
// conversation.
// TODO: make tool use history to infer the flow as a side-quest.
type exportKueryFlowTool struct {
	client clientset.Interface
}

func (t *exportKueryFlowTool) Name() string {
	return "exportKueryFlow"
}

func (t *exportKueryFlowTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name: t.Name(),
			Description: `ExportKueryFlow is a tool that is used to export a KueryFlow from the active conversation.
							A KueryFlow is a CR that contains a sequence of steps. A step can be of type:
							- human: a step requiring human intervention. ONLY contextForHuman field must be set for this type.
							- ai: a step executed by Kuery. ONLY the aiPrompt field must be set for this type.
							- tool: a step executed by a tool. The functionCall field must be set for this type.

							VERY IMPORTANT: Before exporting a KueryFlow, ensure that the user is aware of the steps 
							involved and ask for their agreement first. A step of type 'tool' may involve a function 
							call description with arguments that should be recalculated during later execution.`,

			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the KueryFlow object to be exported.",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "The namespace of the KueryFlow object to be exported.",
					},
					"steps": map[string]interface{}{
						"type":        "array",
						"description": "The steps to export for later execution upon call with ExecuteKueryFlow.",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"description": "The type of the step. Must be one of 'human', 'ai', or 'tool'.",
									"enum":        []string{"human", "ai", "tool"},
								},
								"contextForHuman": map[string]interface{}{
									"type":        "string",
									"description": "A human-readable description of the step for 'human' steps.",
								},
								"aiPrompt": map[string]interface{}{
									"type":        "string",
									"description": "The prompt to send to the AI model for 'ai' steps.",
								},
								"functionCall": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"name": map[string]interface{}{
											"type":        "string",
											"description": "The name of the function to execute.",
										},
										"arguments": map[string]interface{}{
											"type":        "object",
											"description": "Arguments to pass to the function. May include 'RECALCULATE' placeholders.",
										},
										"argsToRecalculate": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "string",
											},
											"description": "List of argument names to recalculate.",
										},
									},
									"required": []string{"name"},
								},
							},
							"required": []string{"type"}, // 'type' is required for every step.
						},
					},
				},
				"required": []string{"name", "namespace", "steps"},
			},
		},
	}
}

type exportCallArgs struct {
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
	Steps     []corev1alpha1.Step `json:"steps"`
}

func (t *exportKueryFlowTool) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args exportCallArgs

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	if err := t.createUpdateKueryFlow(ctx, &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to export KueryFlow: %v", err),
		}
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    "Exported KueryFlow: " + args.Name,
	}
}

func (t *exportKueryFlowTool) createUpdateKueryFlow(ctx context.Context, args *exportCallArgs) error {
	kueryFlow := &corev1alpha1.KueryFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      args.Name,
			Namespace: args.Namespace,
		},
		Spec: corev1alpha1.KueryFlowSpec{
			Steps: args.Steps,
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
