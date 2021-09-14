/*
Copyright 2021.

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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProsodySpec struct {
	DeploymentSpec     `json:",inline"`
	ServiceAnnotations map[string]string `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

// ProsodyStatus defines the observed state of Prosody.
type ProsodyStatus struct {
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Prosody is the Schema for the Prosody API.
type Prosody struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProsodySpec   `json:"spec,omitempty"`
	Status ProsodyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProsodyList contains a list of Prosody.
type ProsodyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Prosody `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Prosody{}, &ProsodyList{})
}
