// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WebSpec struct {
	DeploymentSpec     `json:",inline"`
	ServiceAnnotations map[string]string `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

// WebStatus defines the observed state of Web.
type WebStatus struct {
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Web is the Schema for the Web API.
type Web struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebSpec   `json:"spec,omitempty"`
	Status WebStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WebList contains a list of Web.
type WebList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Web `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Web{}, &WebList{})
}
