package operators_db

import (
	"fmt"
	"github.com/tmc/langchaingo/schema"
)

// getDocs returns a set of operator documents.
// This should be replaced with a scrapper that fetches the data from the web
// and keeps it up to date.
func getDocs() []schema.Document {
	type meta = map[string]any
	docs := []schema.Document{
		{PageContent: "Strimzi provides a way to run an Apache Kafka cluster on Kubernetes or OpenShift in various deployment configurations.",
			Metadata: meta{
				"schema": OperatorSchema{
					Name:       "Strimzi",
					Categories: []string{"OLM Operator", "Streaming and messaging"},
					Features: []string{
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
					},
					Setup: []string{
						`To install the operator, create the following resource:

							apiVersion: operators.coreos.com/v1alpha1
							kind: Subscription
							metadata:
							  name: my-strimzi-kafka-operator
							  namespace: operators
							spec:
							  channel: stable
							  name: strimzi-kafka-operator
							  source: operatorhubio-catalog
							  sourceNamespace: olm`,
					},
				}}},
		{PageContent: "Kubeflow Operator for deployment and management of Kubeflow",
			Metadata: meta{
				"schema": OperatorSchema{
					Name:       "Kubeflow",
					Categories: []string{"OLM Operator", "AI/Machine Learning"},
					Features: []string{
						"Kubeflow is a community and ecosystem of open-source projects to address each stage in the machine learning (ML) lifecycle with support for best-in-class open source tools and frameworks. Kubeflow makes AI/ML on Kubernetes simple, portable, and scalable.",
						"Whether you’re a researcher, data scientist, ML engineer, or a team of developers, Kubeflow offers modular and scalable tools that cater to all aspects of the ML lifecycle: from building ML models to deploying them to production for AI applications.",
					},
					Setup: []string{},
				}}},
		{PageContent: "RabbitMQ is an open source general-purpose message broker that is designed for consistent, highly-available messaging scenarios (both synchronous and asynchronous).",
			Metadata: meta{
				"schema": OperatorSchema{
					Name:       "RabbitMQ",
					Categories: []string{"Helm chart", "Streaming and messaging"},
					Features: []string{
						"Why RabbitMQ? RabbitMQ is a reliable and mature messaging and streaming broker, which is easy to deploy on cloud environments, on-premises, and on your local machine. It is currently used by millions worldwide.",
						"Interoperable: RabbitMQ supports several open standard protocols, including AMQP 1.0 and MQTT 5.0. There are multiple client libraries available, which can be used with your programming language of choice, just pick one. No vendor lock-in!",
						"Flexible: RabbitMQ provides many options you can combine to define how your messages go from the publisher to one or many consumers. Routing, filtering, streaming, federation, and so on, you name it.",
						"Reliable: With the ability to acknowledge message delivery and to replicate messages across a cluster, you can ensure your messages are safe with RabbitMQ.",
					},
					Setup: []string{},
				}}},
		{PageContent: "The RocketMQ Operator manages the Apache RocketMQ service instances deployed on the Kubernetes cluster.",
			Metadata: meta{
				"schema": OperatorSchema{
					Name:       "RocketMQ-Operator",
					Categories: []string{"OLM Operator", "Streaming and messaging"},
					Features: []string{
						"Horizontal Scaling - Safely and seamlessly scale up each component of RocketMQ.",
						"Rolling Update - Gracefully perform rolling updates in order with no downtime.",
						"Multi-cluster Support - Users can deploy and manage multiple RocketMQ name server clusters and broker clusters on a single Kubernetes cluster using RocketMQ Operator.",
						"Topic Transfer - Operator can automatically migrate a specific topic from a source broker cluster to a target cluster without affecting the business.",
					},
					Setup: []string{},
				}}},
		{PageContent: "A Helm chart for creating a KubeStellar Core deployment on a Kubernetes or OpenShift cluster",
			Metadata: meta{
				"schema": OperatorSchema{
					Name:       "KubeStellar",
					Categories: []string{"Helm chart"},
					Features: []string{
						"KubeStellar is a Cloud Native Computing Foundation (CNCF) Sandbox project that simplifies the deployment and configuration of applications across multiple Kubernetes clusters. It provides a seamless experience akin to using a single cluster, and it integrates with the tools you're already familiar with, eliminating the need to modify existing resources.",
						"KubeStellar is particularly beneficial if you're currently deploying in a single cluster and are looking to expand to multiple clusters, or if you're already using multiple clusters and are seeking a more streamlined developer experience.",
					},
					Setup: []string{},
				}}},
	}

	for idx := range docs {
		docs[idx].PageContent = fmt.Sprintf("%s\nMetadata:\n%s",
			docs[idx].PageContent, docs[idx].Metadata["schema"])
	}

	return docs
}
