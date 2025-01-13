package examples

import (
	"context"
	"fmt"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"log"

	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/milvus"
)

func main() {
	store, err := newStore()
	if err != nil {
		log.Fatalf("new: %v\n", err)
	}
	operatorsExample(store)
}

func newStore() (vectorstores.VectorStore, error) {
	llm, err := openai.New(openai.WithBaseURL("http://localhost:5500"))
	if err != nil {
		log.Fatal(err)
	}
	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatal(err)
	}
	idx, err := entity.NewIndexAUTOINDEX(entity.L2)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	milvusConfig := client.Config{
		Address: "http://localhost:19530",
	}
	// Create a new milvus vector store.
	store, errNs := milvus.New(
		ctx,
		milvusConfig,
		milvus.WithDropOld(),
		milvus.WithCollectionName("operators_db"),
		milvus.WithIndex(idx),
		milvus.WithEmbedder(embedder),
	)

	return store, errNs
}

func operatorsExample(store vectorstores.VectorStore) {
	type meta = map[string]any
	// Add documents to the vector store.
	_, errAd := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "Strimzi: Strimzi provides a way to run an Apache Kafka cluster on Kubernetes or OpenShift in various deployment configurations.",
			Metadata: meta{
				"Categories": []string{"OLM Operator", "Streaming and messaging"},
				"Supported Features": []string{
					"Manages the Kafka Cluster - Deploys and manages all of the components of this complex application, including dependencies like Apache ZooKeeper® that are traditionally hard to administer.",
					"Supports KRaft - You can run your Apache Kafka clusters without Apache ZooKeeper.",
					"Tiered storage - Offloads older, less critical data to a lower-cost, lower-performance storage tier, such as object storage.",
					"Includes Kafka Connect - Allows for configuration of common data sources and sinks to move data into and out of the Kafka cluster.",
					"Topic Management - Creates and manages Kafka Topics within the cluster.",
					"User Management - Creates and manages Kafka Users within the cluster.",
					"Connector Management - Creates and manages Kafka Connect connectors.",
					"Includes Kafka Mirror Maker 1 and 2 - Allows for mirroring data between different Apache Kafka® clusters.",
					"Includes HTTP Kafka Bridge - Allows clients to send and receive messages through an Apache Kafka® cluster via HTTP protocol.",
					"Cluster Rebalancing - Uses built-in Cruise Control for redistributing partition replicas according to specified goals in order to achieve the best cluster performance.",
					"Auto-rebalancing when scaling - Automatically rebalance the Kafka cluster after a scale-up or before a scale-down.",
					"Monitoring - Built-in support for monitoring using Prometheus and provided Grafana dashboards",
				}}},
		{PageContent: "Kubeflow: Kubeflow Operator for deployment and management of Kubeflow",
			Metadata: meta{
				"Categories": []string{"OLM Operator", "AI/Machine Learning"},
				"Supported Features": []string{
					"Kubeflow is a community and ecosystem of open-source projects to address each stage in the machine learning (ML) lifecycle with support for best-in-class open source tools and frameworks. Kubeflow makes AI/ML on Kubernetes simple, portable, and scalable.",
					"Whether you’re a researcher, data scientist, ML engineer, or a team of developers, Kubeflow offers modular and scalable tools that cater to all aspects of the ML lifecycle: from building ML models to deploying them to production for AI applications.",
				},
			}},
		{PageContent: "RabbitMQ Operator: The RocketMQ Operator manages the Apache RocketMQ service instances deployed on the Kubernetes cluster.",
			Metadata: meta{
				"Categories": []string{"OLM Operator", "Streaming and messaging"},
				"Supported Features": []string{
					"Horizontal Scaling - Safely and seamlessly scale up each component of RocketMQ.",
					"Rolling Update - Gracefully perform rolling updates in order with no downtime.",
					"Multi-cluster Support - Users can deploy and manage multiple RocketMQ name server clusters and broker clusters on a single Kubernetes cluster using RocketMQ Operator.",
					"Topic Transfer - Operator can automatically migrate a specific topic from a source broker cluster to a target cluster without affecting the business.",
				},
			}},
		{PageContent: "core-chart: A Helm chart for creating a KubeStellar Core deployment on a Kubernetes or OpenShift cluster",
			Metadata: meta{
				"Categories": []string{"Helm chart"},
				"Supported Features": []string{
					"KubeStellar is a Cloud Native Computing Foundation (CNCF) Sandbox project that simplifies the deployment and configuration of applications across multiple Kubernetes clusters. It provides a seamless experience akin to using a single cluster, and it integrates with the tools you're already familiar with, eliminating the need to modify existing resources.",
					"KubeStellar is particularly beneficial if you're currently deploying in a single cluster and are looking to expand to multiple clusters, or if you're already using multiple clusters and are seeking a more streamlined developer experience.",
				},
			}},
	})
	if errAd != nil {
		log.Fatalf("AddDocument: %v\n", errAd)
	}

	ctx := context.Background()
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// run the example cases
	result, err := chains.Run(
		ctx,
		chains.NewRetrievalQAFromLLM(
			llm,
			vectorstores.ToRetriever(store, 2, vectorstores.WithScoreThreshold(0.8)),
		),
		"I wish to add kafka streaming capabilities to my cluster, what operator should I use?",
	)

	fmt.Println(result)
}
