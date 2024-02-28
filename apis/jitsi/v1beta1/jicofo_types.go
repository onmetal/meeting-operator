// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JicofoSpec struct {
	DeploymentSpec `json:",inline"`
	Exporter       Exporter `json:"exporter,omitempty"`
}

// JicofoStatus defines the observed state of JicofoSpec.
type JicofoStatus struct {
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Jicofo is the Schema for the jicodo API.
type Jicofo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JicofoSpec   `json:"spec,omitempty"`
	Status JicofoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JicofoList contains a list of Jicofo.
type JicofoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jicofo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jicofo{}, &JicofoList{})
}
