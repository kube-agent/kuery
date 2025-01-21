package main

import (
	"context"
	"fmt"
	"log"
	"os"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	crd_discovery "github.com/kube-agent/kuery/pkg/crd-discovery"
	"github.com/kube-agent/kuery/pkg/flows"
	"github.com/kube-agent/kuery/pkg/flows/steps"
	operators_db "github.com/kube-agent/kuery/pkg/operators-db"
	"github.com/kube-agent/kuery/pkg/tools"
	"github.com/kube-agent/kuery/pkg/tools/imp"
)

const (
	gptModel       = "gpt-4-1106-preview"
	anthropicModel = "claude-3-5-sonnet-20241022"
)

func setupLLM(ctx context.Context) (llms.Model, error) {
	logger := klog.FromContext(ctx)
	llm := os.Getenv("LLM")

	if llm == "OPENAI" {
		logger.Info("Using OpenAI LLM", "model", gptModel)
		return openai.New(openai.WithModel(gptModel))
	}

	if llm == "ANTHROPIC" {
		logger.Info("Using Anthropic LLM", "model", anthropicModel)
		return anthropic.New(anthropic.WithModel(anthropicModel))
	}

	return nil, fmt.Errorf("LLM provider not set or invalid: %s", llm)
}

func main() {
	ctx := context.Background()
	// init verbosity flag
	klog.InitFlags(nil)

	logger := klog.FromContext(ctx)

	llm, err := setupLLM(ctx)
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := ctrl.GetConfig()
	if err != nil {
		logger.Error(err, "Failed to get kubeconfig, K8s tools won't be enabled")
	}

	toolsMgr := setupToolsMgr(ctx, cfg)
	logger.Info("Tools manager initialized")

	flow := flows.NewConversationalFlow(systemPrompt, llm, toolsMgr, cfg)
	logger.Info("Conversational flow initialized", "tools", toolsMgr.GetToolNames())
	// Sample human step of a user that has a cluster with several services and the need for a high performance message
	// bus operator:
	// I have a cluster with several services and I think I need a high performance message bus operator for event-driven communication.
	flow.HumanStep(steps.ReadFromSTDIN)

	logger.Info("Running flow")

	_, err = flow.Loop(ctx)
	if err != nil {
		logger.Error(err, "Failed to loop flow")
	}
}

func setupToolsMgr(ctx context.Context, cfg *rest.Config) *tools.Manager {
	logger := klog.FromContext(ctx)
	var callables []tools.Tool

	operatorsRetriever, err := operators_db.NewMilvusStore(ctx)
	if err != nil {
		logger.Error(err, "Failed to create operators retriever, tool won't be enabled")
	} else {
		logger.Info("Operators retriever initialized")
		callables = append(callables, &imp.OperatorsRAGTool{operatorsRetriever})
	}

	if cfg != nil {
		dynamicKubeClient, err := dynamic.NewForConfig(cfg)
		if err != nil {
			logger.Error(err, "Failed to create dynamic K8s client, K8s tools won't be enabled")
		} else {
			logger.Info("Dynamic K8s client initialized")
			callables = append(callables, &imp.K8sDynamicClient{dynamicKubeClient})
		}
	}

	apiDiscovery, err := crd_discovery.NewMilvusStore(ctx)
	if err != nil {
		logger.Error(err, "Failed to create API discovery, tool won't be enabled")
	} else {
		logger.Info("API discovery initialized")
		callables = append(callables, &imp.K8sAPIDiscoveryTool{apiDiscovery})
	}

	return tools.NewManager().WithTools(callables)
}

// does not work ATM.
func initHelm(ctx context.Context, cfg *rest.Config, callables *[]tools.Tool) {
	logger := klog.FromContext(ctx)

	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        "default", // Change this to the namespace you wish the client to operate in.
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          true, // Change this to false if you don't want linting.
			DebugLog: func(format string, v ...interface{}) {
				logger.Info(format, v...)
			},
		},
		RestConfig: cfg,
	}

	helmClient, err := helmclient.NewClientFromRestConf(opt)
	if err != nil {
		klog.Error(err, "Failed to create Helm client, Helm tools won't be enabled")
	} else {
		*callables = append(*callables, &imp.HelmTool{helmClient})
	}
}
