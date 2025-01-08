package operators_db

import (
	"context"
	"fmt"
)

// OperatorsRetriever interfaces RAG capabilities to retrieve information.
type OperatorsRetriever interface {
	// Ask asks a question to the RAG model and returns the answer.
	Ask(ctx context.Context, prompt string) (string, error)
	// RetrieveOperators retrieves the OperatorSchema of the operators that are
	// most relevant to the prompt.
	RetrieveOperators(ctx context.Context, prompt string) ([]OperatorSchema, error)
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

// String converts the OperatorSchema to a json string.
func (os *OperatorSchema) String() string {
	// manually build json representation of the struct
	return fmt.Sprintf(`{"name": "%s",\n"categories": %v,\n"features": %v,\n"setup": %v}`,
		os.Name, os.Categories, os.Features, os.Setup)
}

// OperatorSchemasToString converts an array of OperatorSchema to a string.
func OperatorSchemasToString(schemas []OperatorSchema) string {
	array := "["
	for _, schema := range schemas {
		array += "\t\n" + schema.String() + ","
	}

	return array + "\n]"
}
