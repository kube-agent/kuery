package crd_discovery

import (
	"context"
)

// APIDiscovery interfaces RAG capabilities to retrieve custom k8s APIs (CRDs).
type APIDiscovery interface {
	// Ask asks a question to the RAG model and returns the answer.
	Ask(ctx context.Context, prompt string) (string, error)
	// RetrieveCRDs retrieves the CRDs that are most relevant to the prompt.
	RetrieveCRDs(ctx context.Context, prompt string) ([]string, error)
}
