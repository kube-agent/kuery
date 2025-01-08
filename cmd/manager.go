package main

import (
	"context"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/vMaroon/Kuery/pkg/flows"
	"github.com/vMaroon/Kuery/pkg/flows/steps"
	operators_db "github.com/vMaroon/Kuery/pkg/operators-db"
	"github.com/vMaroon/Kuery/pkg/tools"
	"github.com/vMaroon/Kuery/pkg/tools/imp"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"log"
	ctrl "sigs.k8s.io/controller-runtime"
)

const systemPrompt = `
You are a kubernetes and cloud expert that is providing general-purpose assistance for users.
You have access to cluster resources/APIs and sets of operators that can be deployed.

Your access is granted through tool-calling capabilities that wrap APIs and RAG applications. If a tool fails, retry twice max.

You do not only suggest what the user can do, instead you propose doing it for them using the tools you have, after requesting permission.
You extremely prefer to call tools to do the job if they exist in your list of tools.
`

func main() {
	ctx := context.Background()
	// init verbosity flag
	klog.InitFlags(nil)

	logger := klog.FromContext(ctx)

	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("LLM initialized")

	toolsMgr := setupToolsMgr(ctx, llm)
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

func setupToolsMgr(ctx context.Context, llm llms.Model) *tools.Manager {
	operatorsRetriever, err := operators_db.NewMilvusStore(ctx, llm)
	if err != nil {
		log.Fatal(err)
	}

	cfg := ctrl.GetConfigOrDie()

	kubeClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	dynamicKubeClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	return tools.NewManager().WithTools([]tools.Tool{
		&imp.OperatorsRAGTool{
			OperatorsRetriever: operatorsRetriever,
		},
		&imp.K8sCRDsClient{
			Client: kubeClient,
		},
		&imp.K8sDynamicClient{
			Client: dynamicKubeClient,
		},
		&imp.HelmTool{},
	})
}
