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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JicofoSpec struct {
	DeploymentSpec `json:",inline"`
	Exporter       Exporter `json:"exporter,omitempty"`
}

// JicofoStatus defines the observed state of JicofoSpec
type JicofoStatus struct {
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Jicofo is the Schema for the jicodo API
type Jicofo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JicofoSpec   `json:"spec,omitempty"`
	Status JicofoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JicofoList contains a list of Jicofo
type JicofoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jicofo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jicofo{}, &JicofoList{})
}
