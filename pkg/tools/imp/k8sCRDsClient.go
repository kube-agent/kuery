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
	client *apiextensionsclientset.Clientset
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
			Name:        "K8sCRDsClient",
			Description: "Interact with Kubernetes API over CRDs",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type":        "string",
						"description": "The operation to perform: GET", // enough for now
					},
					"resource": map[string]any{
						"type":        "string",
						"description": "The JSON representation of the resource (metav1.Object) to interact with",
					},
				},
				"required": []string{"operation", "resource"},
			},
		},
	}
}

// Call executes the tool call and returns the response.
func (k *K8sCRDsClient) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args struct {
		Operation string `json:"operation"`
		Resource  string `json:"resource"`
	}
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	response, err := k.interactWithK8sCRDs(ctx, args.Operation, args.Resource)
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
func (k *K8sCRDsClient) interactWithK8sCRDs(ctx context.Context, operation, resource string) (string, error) {
	if k.client == nil {
		return "", fmt.Errorf("kubernetes client is not initialized")
	}

	marshalledResource, err := json.Marshal(resource)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resource (%v): %w", resource, err)
	}

	var metaObject metav1.Object
	if err := json.Unmarshal(marshalledResource, &metaObject); err != nil {
		return "", fmt.Errorf("failed to unmarshal resource (%v): %w", resource, err)
	}

	switch operation {
	case "GET":
		crd, err := k.client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, metaObject.GetName(),
			metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get CRD (resource=%v): %w", resource, err)
		}

		return fmt.Sprintf("%v", crd), nil
	}

	return "", fmt.Errorf("unsupported operation: %v", operation)
}
