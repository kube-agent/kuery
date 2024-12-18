package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

// K8sDynamicClient implements the Tool interface for the K8s dynamic client.
type K8sDynamicClient struct {
	client *dynamic.Interface
}

// Name returns the name of the tool.
func (k *K8sDynamicClient) Name() string {
	return "K8sClient"
}

// LLMTool returns the tool as an LLM tool.
func (k *K8sDynamicClient) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name:        "K8sClient",
			Description: "Interact with Kubernetes API goclient",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type":        "string",
						"description": "The operation to perform: GET, POST, PUT, DELETE",
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
func (k *K8sDynamicClient) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
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

	response, err := k.interactWithClient(ctx, args.Operation, args.Resource)
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

// interactWithClient interacts with the Kubernetes API using the go client.
func (k *K8sDynamicClient) interactWithClient(ctx context.Context, operation, resource string) (string, error) {
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
		return fmt.Sprintf("%v", "TODO"), nil
	}

	return "", fmt.Errorf("unsupported operation: %v", operation)
}
