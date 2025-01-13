package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	crd_discovery "github.com/vMaroon/Kuery/pkg/crd-discovery"
)

// K8sAPIDiscoveryTool is a tool that interacts with a vector-db of discovered APIs.
type K8sAPIDiscoveryTool struct {
	crd_discovery.APIDiscovery
}

func (t *K8sAPIDiscoveryTool) Name() string {
	return "clusterAPIDiscoveryTool"
}

func (t *K8sAPIDiscoveryTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name: t.Name(),
			Description: `This tool retrieves the resource most relevant to the user's request.
						  This tool is backed by a K8s API Discovery database that automatically learns the user's custom resources.
						  
							clusterAPIDiscoveryTool should be used to retrieve resource examples in order to fulfill a user request, 
							**typically for interacting with an installed operator**.

							You must call this tool to discover non-core K8s API before using them in the dynamic client.`,

			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type": "string",
						"description": `The prompt to retrieve the CRD for.
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

	crds, err := t.APIDiscovery.RetrieveCRDs(ctx, args.Prompt)
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
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    crd,
	}
}
