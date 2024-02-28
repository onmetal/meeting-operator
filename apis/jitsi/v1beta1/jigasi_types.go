// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JigasiSpec struct {
	DeploymentSpec     `json:",inline"`
	Storage            *StorageSpec      `json:"storage,omitempty"`
	ServiceAnnotations map[string]string `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

// JigasiStatus defines the observed state of Jigasi.
type JigasiStatus struct {
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Jigasi is the Schema for the Jigasi API.
type Jigasi struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JigasiSpec   `json:"spec,omitempty"`
	Status JigasiStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JigasiList contains a list of Jigasi.
type JigasiList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jigasi `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jigasi{}, &JigasiList{})
}
