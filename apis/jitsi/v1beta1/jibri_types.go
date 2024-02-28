// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JibriSpec struct {
	DeploymentSpec     `json:",inline"`
	Storage            *StorageSpec      `json:"storage,omitempty"`
	ServiceAnnotations map[string]string `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

// JibriStatus defines the observed state of JibriSpec.
type JibriStatus struct {
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Jibri is the Schema for the Jibri API.
type Jibri struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JibriSpec   `json:"spec,omitempty"`
	Status JibriStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JibriList contains a list of Jibri.
type JibriList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jibri `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jibri{}, &JibriList{})
}
