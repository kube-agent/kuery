package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// K8sDynamicClient implements the Tool interface for the K8s dynamic client.
type K8sDynamicClient struct {
	Client dynamic.Interface
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
			Name:        "K8sDynamicClient",
			Description: "Interact with Kubernetes API goclient.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type": "string",
						"description": `The operation to perform: LIST, GET, POST, PUT, DELETE
										List resources, get a resource by name, create a resource, update a resource, delete a resource.
										If unsure about the object identification (GVR+namespacedName), use LIST to delegate task to a future call.
										When the intent is to update a resource, use GET to retrieve the resource and then PUT to update it.`,
					},
					"group": map[string]any{
						"type":        "string",
						"description": `The group of the resource to interact with.`,
					},
					"version": map[string]any{
						"type":        "string",
						"description": `The version of the resource to interact with.`,
					},
					"resource": map[string]any{
						"type":        "string",
						"description": `The resource to interact with.`,
					},
					"name": map[string]any{
						"type":        "string",
						"description": `The name of the resource to interact with, or empty for LIST.`,
					},
					"namespace": map[string]any{
						"type":        "string",
						"description": `The namespace of the resource to interact with, or empty for LIST.`,
					},
					"changes": map[string]any{
						"type":        "object",
						"description": `The spec/metadata changes to apply as-is. This would fully replace the existing spec/metadata.`,
						"properties": map[string]any{
							"spec": map[string]any{
								"type":        "object",
								"description": `The spec changes to apply as-is. This would fully replace the existing spec.`,
							},
							"metadata": map[string]any{
								"type":        "object",
								"description": `The metadata changes to apply as-is. This would fully replace the existing metadata.`,
							},
						},
					},
				},
				"required": []string{"operation", "group", "version", "resource", "name", "namespace"},
			},
		},
	}
}

type dynamicCallArgs struct {
	Operation string `json:"operation"`
	Group     string `json:"group"`
	Version   string `json:"version"`
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// Call executes the tool call and returns the response.
func (k *K8sDynamicClient) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args dynamicCallArgs

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	response, err := k.interactWithClient(ctx, args.Operation, args)
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
func (k *K8sDynamicClient) interactWithClient(ctx context.Context, operation string, args dynamicCallArgs) (string, error) {
	if k.Client == nil {
		return "", fmt.Errorf("kubernetes client is not initialized")
	}

	switch operation {
	case "GET":
		var unstructuredObj *unstructured.Unstructured
		var err error

		if args.Namespace == metav1.NamespaceNone {
			unstructuredObj, err = k.Client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).Get(ctx, args.Name, metav1.GetOptions{})
		} else {
			unstructuredObj, err = k.Client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).Namespace(args.Namespace).Get(ctx, args.Name, metav1.GetOptions{})
		}

		if err != nil {
			return "", fmt.Errorf("failed to get resource (namespacedName=%s): %w",
				args.Namespace+"/"+args.Name, err)
		}

		return fmt.Sprintf("%v", unstructuredObj), nil
	}

	return "", fmt.Errorf("unsupported operation: %v", operation)
}
