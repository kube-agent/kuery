package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"

	crd_discovery "github.com/kube-agent/kuery/pkg/crd-discovery"
)

var _ Tool = &K8sAPIDiscoveryTool{}

// K8sAPIDiscoveryTool is a tool that interacts with a vector-db of discovered APIs.
type K8sAPIDiscoveryTool struct {
	retriever crd_discovery.APIDiscovery
}

// NewK8sAPIDiscoveryTool creates a new K8sAPIDiscoveryTool.
func NewK8sAPIDiscoveryTool(retriever crd_discovery.APIDiscovery) *K8sAPIDiscoveryTool {
	return &K8sAPIDiscoveryTool{
		retriever: retriever,
	}
}

func (t *K8sAPIDiscoveryTool) Name() string {
	return "CRExampleFetcher"
}

func (t *K8sAPIDiscoveryTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name: t.Name(),
			Description: `This tool is backed by a K8s API Discovery database that automatically learns the user's custom resources.

							You must call this tool to discover NON-CORE K8s API EXAMPLES (CRs) before using them in the dynamic client.
							Note that this does not install new ones, it only retrieves existing ones. As in, not for installing new operators.

							You should not use this tool for builtin kubernetes resources.`,

			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type": "string",
						"description": `The prompt to retrieve the CR example for.
										This prompt should be generalized and enhanced by you for best results.`,
					},
				},
				"required": []string{"prompt"},
			},
		},
	}
}

func (t *K8sAPIDiscoveryTool) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args struct {
		Prompt string `json:"prompt"`
	}

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	crds, err := t.retriever.RetrieveCRDs(ctx, args.Prompt)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to get relevant CRD: %v", err),
		}
	}

	crd := ""
	for _, c := range crds {
		crd += c
		crd += "\n"
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    crd,
	}
}

// RequiresExplaining returns whether the tool requires explaining after
// execution.
func (t *K8sAPIDiscoveryTool) RequiresExplaining() bool {
	return true
}
