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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JitsiSpec struct {
	Web     `json:"web"`
	Jicofo  `json:"jicofo"`
	JVB     `json:"jvb"`
	Prosody `json:"prosody"`
}

type Web struct {
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/web:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy v1.PullPolicy   `json:"image_pull_policy,omitempty"`
	Environments    []v1.EnvVar     `json:"environments,omitempty"`
	Resources       v1.ResourceList `json:"resources,omitempty"`
	Service         `json:"service,omitempty"`
}

type Jicofo struct {
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/jicofo:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy v1.PullPolicy   `json:"image_pull_policy,omitempty"`
	Environments    []v1.EnvVar     `json:"environments,omitempty"`
	Resources       v1.ResourceList `json:"resources,omitempty"`
	Service         `json:"service,omitempty"`
}

type JVB struct {
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/jvb:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy v1.PullPolicy   `json:"image_pull_policy,omitempty"`
	Environments    []v1.EnvVar     `json:"environments,omitempty"`
	Resources       v1.ResourceList `json:"resources,omitempty"`
	Proxy           `json:"proxy,omitempty"`
	Service         `json:"service,omitempty"`
}

type Prosody struct {
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/prosody:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy v1.PullPolicy   `json:"image_pull_policy,omitempty"`
	Environments    []v1.EnvVar     `json:"environments,omitempty"`
	Resources       v1.ResourceList `json:"resources,omitempty"`
	Service         `json:"service,omitempty"`
}

type Proxy struct {
	//+kubebuilder:default="haproxy:2.2.11"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	Service  `json:"service,omitempty"`
}

// JitsiStatus defines the observed state of Jitsi
type JitsiStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Jitsi is the Schema for the jitsis API
type Jitsi struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JitsiSpec   `json:"spec,omitempty"`
	Status JitsiStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JitsiList contains a list of Jitsi
type JitsiList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jitsi `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jitsi{}, &JitsiList{})
}
