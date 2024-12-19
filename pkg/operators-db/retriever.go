package operators_db

import (
	"context"
)

// OperatorsRetriever interfaces RAG capabilities to retrieve information.
type OperatorsRetriever interface {
	// Ask asks a question to the RAG model and returns the answer.
	Ask(ctx context.Context, prompt string) (string, error)
	// RetrieveOperator retrieves the OperatorSchema of the operator that is
	// most relevant to the prompt.
	RetrieveOperator(ctx context.Context, prompt string) (*OperatorSchema, error)
}

// OperatorSchema defines the schema for an operator.
// The schema is similar to that defined in helm/OLM operator repositories.
type OperatorSchema struct {
	// Name of the operator.
	Name string `json:"name"`
	// Categories the operator falls under.
	Categories []string `json:"categories"`
	// Features of the operator.
	Features []string `json:"features"`
	// Setup contains the setup documentation for the operator.
	Setup []string `json:"setup"`
}
