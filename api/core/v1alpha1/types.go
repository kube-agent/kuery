/*
Copyright 2025 The Kuery Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/tmc/langchaingo/llms"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KueryFlow specifies a sequence of Steps to be executed in order.
//
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=kf
type KueryFlow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KueryFlowSpec   `json:"spec,omitempty"`
	Status KueryFlowStatus `json:"status,omitempty"`
}

// KueryFlowSpec defines the desired state of KueryFlow.
type KueryFlowSpec struct {
	// steps is a sequence of steps to be executed in order.
	Steps []Step `json:"steps"`
}

// Step defines a step in a KueryFlow.
type Step struct {
	// type is the type of the step.
	// A step can be of types:
	// - human: a step that requires human intervention. Typically, this
	// means that the step is a manual step that requires a human to
	// input a message to Kuery.
	// When type is `human`, the contextForHuman field should be set.
	//
	// - ai: a step that is executed by an AI model. This is a step that
	// is executed by an AI model, and the result is returned to Kuery.
	// When type is `ai`, the aiPrompt field MUST be set.
	//
	// - tool: a step that is executed by a tool. This is a step that
	// results in a function call to a tool. When type is `tool`, the
	// functionCall field MUST be set.
	// +kubebuilder:validation:Enum=human;ai;tool
	Type string `json:"type"`

	// contextForHuman is a human-readable description of the step, to be
	// displayed in the UI in the case of a human step.
	ContextForHuman *string `json:"contextForHuman,omitempty"`

	// aiPrompt is the prompt to be sent to the AI model in an AI step.
	AIPrompt *string `json:"aiPrompt,omitempty"`

	// functionCall is the function call to be executed in a tool step.
	// A functionCall consists of the name of the function to be executed,
	// and the parameters to be passed to the function. A parameter may be
	// a concrete value or "RECALCULATE" to indicate that the value should
	// be recalculated by the Kuery.
	FunctionCall *llms.FunctionCall `json:"functionCall,omitempty"`
	// argsToRecalculate is a list of argument-names that should be
	// recalculated by the Kuery. Names appearing in this list should be
	// present in the parameters of the functionCall.
	ArgsToRecalculate []string `json:"argsToRecalculate,omitempty"`
}

const (
	// StepTypeHuman is the type of human step.
	StepTypeHuman = "human"
	// StepTypeAI is the type of AI step.
	StepTypeAI = "ai"
	// StepTypeTool is the type of tool step.
	StepTypeTool = "tool"
)

// KueryFlowStatus defines the observed state of KueryFlow.
type KueryFlowStatus struct {
}

// +kubebuilder:object:root=true

// KueryFlowList contains a list of KueryFlow.
type KueryFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KueryFlow `json:"items"`
}
