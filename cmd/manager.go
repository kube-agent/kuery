package main

import (
	"context"
	"fmt"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
	crd_discovery "github.com/vMaroon/Kuery/pkg/crd-discovery"
	"github.com/vMaroon/Kuery/pkg/flows"
	"github.com/vMaroon/Kuery/pkg/flows/steps"
	operators_db "github.com/vMaroon/Kuery/pkg/operators-db"
	"github.com/vMaroon/Kuery/pkg/tools"
	"github.com/vMaroon/Kuery/pkg/tools/imp"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"log"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	gptModel       = "gpt-4-1106-preview"
	anthropicModel = "claude-3-5-sonnet-20241022"
	systemPrompt   = `
You are a kubernetes and cloud expert that is providing general-purpose assistance for users.
You have access to cluster resources/APIs and sets of operators that can be deployed. The user does not see toolcalls, make sure to be transparent about it.

Your access is granted through tool-calling capabilities that wrap APIs. If a tool fails, retry twice max.

NOTE THAT you have the unique "addContext" tool to forcefully grant your self an additional turn before the user.'
You should use "addContext" if resolving a user's request requires multi-step planning or added execution.

You do not only suggest what the user can do, instead you propose doing it for them using the tools you have after requesting permission.
You extremely prefer to call tools to do the job if they exist in your list of tools.

Make sure the user agrees with what you're doing, especially before cluster-effecting tool calls.
When running multi-step plans, make sure to ask the user for permission before every step.'
`
)

func setupLLM(ctx context.Context) (llms.Model, error) {
	logger := klog.FromContext(ctx)
	llm := os.Getenv("LLM")
	// check if openai api key is set
	if llm == "OPENAI" {
		logger.Info("Using OpenAI LLM", "model", gptModel)
		return openai.New(openai.WithModel(gptModel))
	}

	if llm == "ANTHROPIC" {
		logger.Info("Using Anthropic LLM", "model", anthropicModel)
		return anthropic.New(anthropic.WithModel(anthropicModel))
	}

	return nil, fmt.Errorf("no API key found")
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

	toolsMgr := setupToolsMgr(ctx)
	logger.Info("Tools manager initialized")

	flow := flows.NewConversationalFlow(systemPrompt, llm, toolsMgr)
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

func setupToolsMgr(ctx context.Context) *tools.Manager {
	logger := klog.FromContext(ctx)
	var callables []tools.Tool

	operatorsRetriever, err := operators_db.NewMilvusStore(ctx)
	if err != nil {
		logger.Error(err, "Failed to create operators retriever, tool won't be enabled")
	} else {
		logger.Info("Operators retriever initialized")
		callables = append(callables, &imp.OperatorsRAGTool{operatorsRetriever})
	}

	cfg, err := ctrl.GetConfig()
	if err != nil {
		logger.Error(err, "Failed to get kubeconfig, K8s tools won't be enabled")
	} else {
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
