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

package v1alpha1

import (
	"errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	TypeAssertionError = errors.New("type assertion failure")
)

// EtherpadSpec defines the desired state of Etherpad
type EtherpadSpec struct {
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="etherpad/etherpad:1.8.12"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy v1.PullPolicy   `json:"image_pull_policy,omitempty"`
	Environments    []v1.EnvVar     `json:"environments,omitempty"`
	Resources       v1.ResourceList `json:"resources,omitempty"`
	//+kubebuilder:default="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Services    []Service      `json:"service,omitempty"`
}

type Service struct {
	//+kubebuilder:default:="ClusterIP"
	Type     v1.ServiceType `json:"type,omitempty"`
	Protocol v1.Protocol    `json:"protocol,omitempty"`
	//+kubebuilder:default="http"
	PortName string `json:"port_name,omitempty"`
	//+kubebuilder:default=9001
	Port int32 `json:"port,omitempty"`
}

// EtherpadStatus defines the observed state of Etherpad
type EtherpadStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Etherpad is the Schema for the etherpads API
type Etherpad struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtherpadSpec   `json:"spec,omitempty"`
	Status EtherpadStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EtherpadList contains a list of Etherpad
type EtherpadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Etherpad `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Etherpad{}, &EtherpadList{})
}
