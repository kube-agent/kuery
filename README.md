# Kuery

#### Disclaimer: Project is currently in a PoC state.
Kuery is an open-source LLM-based assistant that allows users to chat with their Kubernetes clusters. 
Kuery can assist users with deploying applications, managing resources, and much more.

Kuery enables Kubernetes users by bridging knowledge gaps and simplifying the interaction with the cluster, 
and empowers power users by accelerating and automating their workflows.

## System

Kuery provides a turn-based terminal chat interface that allows you to interact with your Kubernetes cluster. 
It is equipped with several tools that can help you manage your cluster more efficiently:
1. Operators Discovery: Kuery is backed by an operators database that allows it to fit and install the best operator for your needs
2. Cluster Discovery: Kuery also maintains a database of cluster APIs and resources to accurately manage your resources
3. API Server: Kuery can interact with your cluster's API server to manage resources and deploy applications

The discovery components are currently in demo state, they pend proper implementation and integration with the system.

In addition to the above capabilities, Kuery will be able extract workflows from your chat into a custom resource that can be used to automate tasks.

## Use Cases

1. Any user who wants to interact with their Kubernetes cluster in a conversational manner
2. Users who want to deploy applications and manage resources without having to remember complex commands, or are unaware of the possibilities
3. Users wishing to automate tasks and workflows in their Kubernetes cluster

## Installation (PoC)

### Configuration

#### LLM

1. OPENAI (gpt-4-1106-preview)
```
    export LLM=OPENAI
    export MODEL=gpt-4-1106-preview
    export OPENAI_API_KEY=...
```

2. ANTHROPIC (claude-3-5-sonnet-20241022)
```
    export LLM=ANTHROPIC
    export MODEL=claude-3-5-sonnet-20241022
    export ANTHROPIC_API_KEY=...
```

#### Kubernetes

Set the `KUBECONFIG` environment variable to the path of your kubeconfig file
```
    export KUBECONFIG=~/.kube/config
```

#### Milvus for Vector DBs

```
    cd pkg/operators-db && docker-compose up -d
```