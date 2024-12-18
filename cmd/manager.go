package main

import (
	"context"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/vMaroon/Kuery/pkg/flows"
	operators_db "github.com/vMaroon/Kuery/pkg/operators-db"
	"github.com/vMaroon/Kuery/pkg/tools"
	"github.com/vMaroon/Kuery/pkg/tools/imp"
	"k8s.io/klog/v2"
	"log"
)

func main() {
	ctx := context.Background()
	klog.InitFlags(nil)
	logger := klog.FromContext(ctx)

	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("LLM initialized")

	toolsMgr := setupToolsMgr(ctx, llm)
	logger.Info("Tools manager initialized")

	flow := flows.NewConversationalFlow(llm, toolsMgr)
	flow.HumanStep("I wish to add message streaming (or pub/sub) capabilities to my cluster. What should I do?")

	logger.Info("Running flow")
	history, err := flow.Run(ctx)
	if err != nil {
		logger.Error(err, "failed to run flow")
	}

	for _, msg := range history {
		logger.Info("Step", "role", msg.Role, "content", msg.Parts)
	}
}

func setupToolsMgr(ctx context.Context, llm llms.Model) *tools.Manager {
	operatorsRetriever, err := operators_db.NewMilvusStore(ctx, llm)
	if err != nil {
		log.Fatal(err)
	}

	return tools.NewManager().WithTool(&imp.OperatorsTool{
		OperatorsRetriever: operatorsRetriever,
	})
}
