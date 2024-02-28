// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
