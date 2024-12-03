package operators_db

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"log"

	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/milvus"
)

const (
	baseURL = "http://localhost:5500"
)

// MilvusStore implements OperatorsRetriever with Milvus as vector DB backend.
type MilvusStore struct {
	store          vectorstores.VectorStore
	retrievalChain chains.RetrievalQA
}

// NewMilvusStore creates a new MilvusStore instance.
func NewMilvusStore(ctx context.Context, llm llms.Model) (OperatorsRetriever, error) {
	openaiLLM, err := openai.New(openai.WithBaseURL(baseURL))
	if err != nil {
		log.Fatal(err)
	}
	embedder, err := embeddings.NewEmbedder(openaiLLM)
	if err != nil {
		log.Fatal(err)
	}
	idx, err := entity.NewIndexAUTOINDEX(entity.L2)
	if err != nil {
		log.Fatal(err)
	}

	milvusConfig := client.Config{
		Address: "http://localhost:19530",
	}
	// Create a new milvus vector store.
	store, err := milvus.New(
		ctx,
		milvusConfig,
		milvus.WithDropOld(),
		milvus.WithCollectionName("operators_db"),
		milvus.WithIndex(idx),
		milvus.WithEmbedder(embedder),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus store: %w", err)
	}
	// populate docs
	if err := populateDocs(ctx, store); err != nil {
		return nil, fmt.Errorf("failed to populate docs: %w", err)
	}

	return &MilvusStore{
		store: store,
		retrievalChain: chains.NewRetrievalQAFromLLM(llm,
			vectorstores.ToRetriever(store, 3, vectorstores.WithScoreThreshold(0.6))),
	}, nil
}

// Ask asks a question to the RAG model and returns the answer.
func (m *MilvusStore) Ask(ctx context.Context, prompt string) (string, error) {
	result, err := chains.Run(
		ctx,
		m.retrievalChain,
		prompt,
	)

	if err != nil {
		return "", fmt.Errorf("failed to run retrieval chain: %w", err)
	}

	return result, nil
}

// RetrieveOperator retrieves the OperatorSchema of the operator that is
// most relevant to the prompt.
func (m *MilvusStore) RetrieveOperator(ctx context.Context, prompt string) (*OperatorSchema, error) {
	templatedPrompt := fmt.Sprintf("What is the name of the operator that is most relevant to the following user"+
		"prompt? /nUser prompt: %s", prompt)
	name, err := m.Ask(ctx, templatedPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get operator name: %w", err)
	}

	docs, err := m.store.SimilaritySearch(ctx, name, 1, vectorstores.WithScoreThreshold(0.8))
	if err != nil {
		return nil, fmt.Errorf("failed to get relevant documents: %w", err)
	}

	if len(docs) == 0 {
		return nil, nil
	}

	return unstructuredMapToOperatorSchema(docs[0].Metadata["schema"].(map[string]interface{}))
}

func populateDocs(ctx context.Context, store vectorstores.VectorStore) error {
	if _, err := store.AddDocuments(context.Background(), getDocs()); err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	return nil
}

func unstructuredMapToOperatorSchema(m map[string]interface{}) (*OperatorSchema, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map: %w", err)
	}

	operatorSchema := OperatorSchema{}
	if err := json.Unmarshal(data, &operatorSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operator schema: %w", err)
	}

	return &operatorSchema, nil
}
