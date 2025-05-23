package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kube-agent/kuery/pkg/tools/api"

	"github.com/tmc/langchaingo/llms"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var _ api.Tool = &K8sDynamicClient{}

// K8sDynamicClient implements the Tool interface for the K8s dynamic client.
type K8sDynamicClient struct {
	client dynamic.Interface
}

// NewK8sDynamicClient creates a new K8sDynamicClient.
func NewK8sDynamicClient(client dynamic.Interface) *K8sDynamicClient {
	return &K8sDynamicClient{
		client: client,
	}
}

// Name returns the name of the tool.
func (k *K8sDynamicClient) Name() string {
	return "K8sDynamicClient"
}

// LLMTool returns the tool as an LLM tool.
func (k *K8sDynamicClient) LLMTool() *llms.Tool {
	desc := `Interact with Kubernetes Cluster.
			This tool should be used when the user wants to interact with the Kubernetes cluster.`

	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name:        k.Name(),
			Description: api.AddApprovalRequirementToDescription(k, desc),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type": "string",
						"description": `The operation to perform: LIST, GET, POST, PUT, DELETE
										LIST: List resources.
										GET: Get a resource by name.	
										POST: Create a resource. When using POST, you can skip identification fields (except for NS if needed).
										PUT: UPDATE a resource. When using PUT, you can skip identification fields (except for NS if needed).
										DELETE: Delete a resource.
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
					"object": map[string]any{
						"type":        "string",
						"description": `The object to create or update. This should be a JSON object that can be unmarshalled in go.`,
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

	Object string `json:"object"`
}

// Call executes the tool call and returns the response.
func (k *K8sDynamicClient) Call(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool) {
	var args dynamicCallArgs

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}, false
	}

	response, err := k.interactWithClient(ctx, args.Operation, args)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to interact with Kubernetes API: %v", err),
		}, false
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    response,
	}, true
}

// interactWithClient interacts with the Kubernetes API using the go client.
func (k *K8sDynamicClient) interactWithClient(ctx context.Context, operation string, args dynamicCallArgs) (string, error) {
	if k.client == nil {
		return "", fmt.Errorf("kubernetes client is not initialized")
	}

	switch operation {
	case "GET":
		var unstructuredObj *unstructured.Unstructured
		var err error

		if args.Namespace == metav1.NamespaceNone {
			unstructuredObj, err = k.client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).Get(ctx, args.Name, metav1.GetOptions{})
		} else {
			unstructuredObj, err = k.client.Resource(schema.GroupVersionResource{
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
	case "LIST":
		var unstructuredList *unstructured.UnstructuredList
		var err error

		if args.Namespace == metav1.NamespaceNone {
			unstructuredList, err = k.client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).List(ctx, metav1.ListOptions{})
		} else {
			unstructuredList, err = k.client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).Namespace(args.Namespace).List(ctx, metav1.ListOptions{})
		}

		if err != nil {
			return "", fmt.Errorf("failed to list resources: %w", err)
		}

		return fmt.Sprintf("%v", unstructuredList), nil
	case "POST":
		fallthrough
	case "PUT":
		var err error
		var unstructuredObj *unstructured.Unstructured

		if err := json.Unmarshal([]byte(args.Object), &unstructuredObj); err != nil {
			return "", fmt.Errorf("failed to unmarshal object: %w", err)
		}

		if args.Namespace == metav1.NamespaceNone {
			unstructuredObj, err = k.client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).Create(ctx, unstructuredObj, metav1.CreateOptions{})
		} else {
			unstructuredObj, err = k.client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).Namespace(args.Namespace).Create(ctx, unstructuredObj, metav1.CreateOptions{})
		}

		if err != nil {
			return "", fmt.Errorf("failed to create resource: %w", err)
		}

		return fmt.Sprintf("%v", unstructuredObj), nil
	case "DELETE":
		var err error

		if args.Namespace == metav1.NamespaceNone {
			err = k.client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).Delete(ctx, args.Name, metav1.DeleteOptions{})
		} else {
			err = k.client.Resource(schema.GroupVersionResource{
				Group:    args.Group,
				Version:  args.Version,
				Resource: args.Resource,
			}).Namespace(args.Namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
		}

		if err != nil {
			return "", fmt.Errorf("failed to delete resource: %w", err)
		}
	}

	return "", fmt.Errorf("unsupported operation: %v", operation)
}

// RequiresExplaining returns whether the tool requires explaining after
// execution.
func (k *K8sDynamicClient) RequiresExplaining() bool {
	return true
}

// RequiresApproval returns whether the tool requires approval before
// execution.
func (k *K8sDynamicClient) RequiresApproval() bool { return true }
