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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KueryFlow specifies a sequence of Steps to be executed in order.
//
// +genclient
// +genclient:namespaced
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
	// TODO
}

// KueryFlowStatus defines the observed state of KueryFlow.
type KueryFlowStatus struct {
	// TODO
}

// +kubebuilder:object:root=true

// KueryFlowList contains a list of KueryFlow.
type KueryFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KueryFlow `json:"items"`
}
