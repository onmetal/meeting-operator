// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JVBSpec struct {
	DeploymentSpec     `json:",inline"`
	Exporter           Exporter          `json:"exporter,omitempty"`
	CustomSIP          []string          `json:"custom_sip,omitempty"`
	ServiceAnnotations map[string]string `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Port        Port           `json:"port,omitempty"`
}

// JVBStatus defines the observed state of JVBSpec.
type JVBStatus struct {
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=".status.replicas"

// JVB is the Schema for the JVB API.
type JVB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JVBSpec   `json:"spec,omitempty"`
	Status JVBStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JVBList contains a list of JVB.
type JVBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JVB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JVB{}, &JVBList{})
}
