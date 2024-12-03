package main

import (
	"context"
	"github.com/tmc/langchaingo/llms/openai"
	operators_db "github.com/vMaroon/Kuery/pkg/operators-db"
	"log"
)

func main() {
	ctx := context.Background()
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	operatorsRetriever, err := operators_db.NewMilvusStore(ctx, llm)
	if err != nil {
		log.Fatal(err)
	}

	operator, err := operatorsRetriever.RetrieveOperator(ctx,
		"I wish to add message streaming capabilities to my cluster, what operator should I use?")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Operator: %v\n", operator)
}
