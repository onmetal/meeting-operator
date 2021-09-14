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

package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EtherpadSpec defines the desired state of Etherpad.
type EtherpadSpec struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="etherpad/etherpad:1.8.12"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy    v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets   []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	SecurityContext    v1.SecurityContext        `json:"security_context,omitempty"`
	Environments       []v1.EnvVar               `json:"environments,omitempty"`
	Resources          v1.ResourceRequirements   `json:"resources,omitempty"`
	ServiceAnnotations map[string]string         `json:"service_annotations,omitempty"`
	//+kubebuilder:default="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

type Port struct {
	//+kubebuilder:default="http"
	Name     string      `json:"name,omitempty"`
	Protocol v1.Protocol `json:"protocol,omitempty"`
	Port     int32       `json:"port,omitempty"`
}

// EtherpadStatus defines the observed state of Etherpad.
type EtherpadStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Etherpad is the Schema for the etherpads API.
type Etherpad struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtherpadSpec   `json:"spec,omitempty"`
	Status EtherpadStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EtherpadList contains a list of Etherpad.
type EtherpadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Etherpad `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Etherpad{}, &EtherpadList{})
}
