package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kube-agent/kuery/pkg/tools/api"

	"github.com/tmc/langchaingo/llms"

	crd_discovery "github.com/kube-agent/kuery/pkg/crd-discovery"
)

var _ api.Tool = &K8sAPIDiscoveryTool{}

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
	desc := `This tool is used to learn about custom-resources (non-builtin resources!) before interacting with them within the dynamic client.
				You MUST NOT use this tool for builtin kubernetes resources such as deployments, pods...`

	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name:        t.Name(),
			Description: api.AddApprovalRequirementToDescription(t, desc),
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

func (t *K8sAPIDiscoveryTool) Call(ctx context.Context, toolCall *llms.ToolCall) (llms.ToolCallResponse, bool) {
	var args struct {
		Prompt string `json:"prompt"`
	}

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}, false
	}

	crds, err := t.retriever.RetrieveCRDs(ctx, args.Prompt)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to get relevant CRD: %v", err),
		}, false
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
	}, true
}

// RequiresExplaining returns whether the tool requires explaining after
// execution.
func (t *K8sAPIDiscoveryTool) RequiresExplaining() bool {
	return true
}

// RequiresApproval returns whether the tool requires approval before
// execution.
func (t *K8sAPIDiscoveryTool) RequiresApproval() bool { return false }
