package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sCRDsClient implements the Tool interface for the Kubernetes API over CRDs.
type K8sCRDsClient struct {
	Client *apiextensionsclientset.Clientset
}

// Name returns the name of the tool.
func (k *K8sCRDsClient) Name() string {
	return "K8sCRDsClient"
}

// LLMTool returns the tool as an LLM tool.
func (k *K8sCRDsClient) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name: "K8sCRDsClient",
			Description: `Interact with Kubernetes Cluster over CRD resources.
						  This tool should be used when the user wants to interact with the Kubernetes cluster, specifically with CRD resources.
							This tool should be used with clear user intent.`,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type": "string",
						"description": `The operation to perform: GET, LIST.
										Get a CRD by name or list all CRDs.
										If unsure about the resource field, use LIST and delegate task to a future call.`,
					},
					"name": map[string]any{
						"type":        "string",
						"description": `The name of the CRD to interact with, or empty for LIST.`,
					},
				},
				"required": []string{"operation", "name"},
			},
		},
	}
}

// Call executes the tool call and returns the response.
func (k *K8sCRDsClient) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args struct {
		Operation string `json:"operation"`
		Name      string `json:"name"`
	}
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	response, err := k.interactWithK8sCRDs(ctx, args.Operation, args.Name)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to interact with Kubernetes API: %v", err),
		}
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    response,
	}
}

// interactWithK8sCRDs interacts with the Kubernetes API over CRDs.
func (k *K8sCRDsClient) interactWithK8sCRDs(ctx context.Context, operation, name string) (string, error) {
	if k.Client == nil {
		return "", fmt.Errorf("kubernetes client is not initialized")
	}

	switch operation {
	case "GET":
		crd, err := k.Client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get CRD (name=%s): %w", name, err)
		}

		return fmt.Sprintf("%v", crd), nil
	case "LIST":
		crds, err := k.Client.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list CRDs: %w", err)
		}

		return fmt.Sprintf("%v", crds), nil
	}

	return "", fmt.Errorf("unsupported operation: %v", operation)
}
