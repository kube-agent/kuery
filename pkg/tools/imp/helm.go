package imp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	"helm.sh/helm/v3/pkg/repo"
	"time"

	"github.com/mittwald/go-helm-client"
)

type HelmTool struct {
	Client helmclient.Client
}

func (ht *HelmTool) Name() string {
	return "helmTool"
}

func (ht *HelmTool) LLMTool() *llms.Tool {
	return &llms.Tool{
		Type: functionToolType,
		Function: &llms.FunctionDefinition{
			Name: ht.Name(),
			Description: "Interact with Helm. Add chart repo, install chart, delete chart, upgrade chart." +
				"This tool should be used with clear user intent.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type": "string",
						"description": `The operation to perform:
										ADD_REPO: adds a Helm chart repository
													For this operation, use the 'name' and 'url' fields.
										INSTALL: installs a Helm chart
													For this operation, use the 'releaseName', 'chartName' and 'namespace' fields.
										DELETE: deletes a Helm chart. Fields are same as INSTALL.
										UPGRADE: upgrades a Helm chart. Fields are same as ADD_REPO.`,
					},
					"name": map[string]any{
						"type":        "string",
						"description": "The name of the Helm chart repository. For example: bitnami",
					},
					"url": map[string]any{
						"type":        "string",
						"description": "The URL of the Helm chart repository. For example: https://charts.bitnami.com/bitnami",
					},
					"releaseName": map[string]any{
						"type":        "string",
						"description": "The release name of the Helm chart. For example: etcd-operator",
					},
					"chartName": map[string]any{
						"type":        "string",
						"description": "The name of the Helm chart. For example: stable/etcd-operator",
					},
				},
				"required": []string{"operation"},
			},
		},
	}
}

type chartCall struct {
	Operation   string `json:"operation"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	ReleaseName string `json:"releaseName"`
	ChartName   string `json:"chartName"`
	Namespace   string `json:"namespace"`
}

func (ht *HelmTool) Call(ctx context.Context, toolCall *llms.ToolCall) llms.ToolCallResponse {
	var args chartCall

	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to unmarshal arguments: %v", err),
		}
	}

	response, err := ht.helmOperation(ctx, args)
	if err != nil {
		return llms.ToolCallResponse{
			ToolCallID: toolCall.ID,
			Name:       toolCall.FunctionCall.Name,
			Content:    fmt.Sprintf("failed to execute Helm operation: %v", err),
		}
	}

	return llms.ToolCallResponse{
		ToolCallID: toolCall.ID,
		Name:       toolCall.FunctionCall.Name,
		Content:    response,
	}
}

// helmOperation executes the Helm operation and returns the response.
func (ht *HelmTool) helmOperation(ctx context.Context, args chartCall) (string, error) {
	switch args.Operation {
	case "ADD_REPO":
		if err := ht.Client.AddOrUpdateChartRepo(repo.Entry{
			Name: args.Name,
			URL:  args.URL,
		}); err != nil {
			return "", fmt.Errorf("failed to add Helm chart repository: %w", err)
		}

		return fmt.Sprintf("Helm chart repository %q added successfully", args.Name), nil
	case "INSTALL":
		release, err := ht.Client.InstallChart(ctx, &helmclient.ChartSpec{
			ReleaseName: args.ReleaseName,
			ChartName:   args.ChartName,
			Namespace:   args.Namespace,
			UpgradeCRDs: true,
			Wait:        true,
			Timeout:     32 * time.Second,
		}, nil)

		if err != nil {
			return "", fmt.Errorf("failed to install Helm chart: %w", err)
		}

		return fmt.Sprintf("Helm chart %q installed successfully", release.Name), nil
	default:
		return "", fmt.Errorf("unsupported operation %q", args.Operation)
	}
}
