package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/vMaroon/Kuery/pkg/flows"
	operators_db "github.com/vMaroon/Kuery/pkg/operators-db"
	"github.com/vMaroon/Kuery/pkg/tools"
	"github.com/vMaroon/Kuery/pkg/tools/imp"
	"k8s.io/klog/v2"
	"log"
	"os"
)

const systemPrompt = `
You are a kubernetes and cloud expert that is providing general-purpose assistance for users.
You have access to cluster resources/APIs and sets of operators that can be deployed.

Your access is granted through tool-calling capabilities that wrap APIs and RAG applications.

You do not only suggest what the user can do, instead you propose doing it for them using the tools you have, after requesting permission.
You extremely prefer to call tools to do the job if they exist in your list of tools.
`

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

	flow := flows.NewConversationalFlow(systemPrompt, llm, toolsMgr)
	flow.HumanStep(func(_ context.Context) string {
		// I wish to add message streaming (or pub/sub) capabilities to my cluster. What should I do?
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("User Input: ")
		text, _ := reader.ReadString('\n')
		return text
	})

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

	return tools.NewManager().WithTools([]tools.Tool{
		&imp.OperatorsRAGTool{
			OperatorsRetriever: operatorsRetriever,
		},
		&imp.K8sCRDsClient{},
		&imp.K8sDynamicClient{},
		&imp.HelmTool{},
	})
}
