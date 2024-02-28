// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WhiteBoardSpec defines the desired state of WhiteBoard.
type WhiteBoardSpec struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="excalidraw/excalidraw:sha-5c73c58"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy    v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets   []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	SecurityContext    v1.SecurityContext        `json:"security_context,omitempty"`
	Environments       []v1.EnvVar               `json:"environments,omitempty"`
	Resources          v1.ResourceRequirements   `json:"resources,omitempty"`
	ServiceAnnotations map[string]string         `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

type Port struct {
	//+kubebuilder:default="http"
	Name     string      `json:"name,omitempty"`
	Protocol v1.Protocol `json:"protocol,omitempty"`
	Port     int32       `json:"port,omitempty"`
}

// WhiteBoardStatus defines the observed state of WhiteBoard.
type WhiteBoardStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type WhiteBoard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WhiteBoardSpec   `json:"spec,omitempty"`
	Status WhiteBoardStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type WhiteBoardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WhiteBoard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WhiteBoard{}, &WhiteBoardList{})
}
