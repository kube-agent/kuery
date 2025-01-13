package crd_discovery

import (
	"fmt"
	"github.com/tmc/langchaingo/schema"
)

// getDocs returns a list of CRDs for demo purposes.
// Currently this is a hardcoded list of strimzi installed CRDs.
// This should be replaced with a scrapper that fetches the data from the web
// and keeps it up to date.
func getDocs() []schema.Document {
	type meta = map[string]any
	docs := []schema.Document{
		{PageContent: "KafkaTopic: resource for creating Topics in a Kafka instance.",
			Metadata: meta{
				"cr_example": `apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: my-topic
  labels:
    strimzi.io/cluster: my-cluster
spec:
  partitions: 1
  replicas: 1
  config:
    retention.ms: 7200000
    segment.bytes: 1073741824`,
			}},
		{PageContent: "Kafka: resource for deploying the Kafka instance.",
			Metadata: meta{
				"cr_example": `apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: my-cluster
spec:
  kafka:
    version: 3.9.0
    replicas: 1
    listeners:
      - name: plain
        port: 9092
        type: internal
        tls: false
      - name: tls
        port: 9093
        type: internal
        tls: true
    config:
      offsets.topic.replication.factor: 1
      transaction.state.log.replication.factor: 1
      transaction.state.log.min.isr: 1
      default.replication.factor: 1
      min.insync.replicas: 1
      inter.broker.protocol.version: "3.9"
    storage:
      type: ephemeral
  zookeeper:
    replicas: 3
    storage:
      type: ephemeral
  entityOperator:
    topicOperator: {}
    userOperator: {}`,
			}},
	}

	for idx := range docs {
		docs[idx].PageContent = fmt.Sprintf("%s\n CR Example: %s",
			docs[idx].PageContent, docs[idx].Metadata["cr_example"])
	}

	return docs
}
