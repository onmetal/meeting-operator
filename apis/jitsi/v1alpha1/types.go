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
	Prosody `json:"prosody"`
	Jicofo  `json:"jicofo"`
	Jibri   `json:"jibri"`
	JVB     `json:"jvb"`
}

type Web struct {
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/web:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy    v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets   []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	Environments       []v1.EnvVar               `json:"environments,omitempty"`
	Resources          v1.ResourceRequirements   `json:"resources,omitempty"`
	ServiceAnnotations map[string]string         `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

type Prosody struct {
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/prosody:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy    v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets   []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	Environments       []v1.EnvVar               `json:"environments,omitempty"`
	Resources          v1.ResourceRequirements   `json:"resources,omitempty"`
	ServiceAnnotations map[string]string         `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

type Jicofo struct {
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/jicofo:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy    v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets   []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	Environments       []v1.EnvVar               `json:"environments,omitempty"`
	Resources          v1.ResourceRequirements   `json:"resources,omitempty"`
	ServiceAnnotations map[string]string         `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

type Jibri struct {
	//+kubebuilder:default:=0
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/jibri:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy    v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets   []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	Environments       []v1.EnvVar               `json:"environments,omitempty"`
	Storage            *StorageSpec              `json:"storage,omitempty"`
	Resources          v1.ResourceRequirements   `json:"resources,omitempty"`
	ServiceAnnotations map[string]string         `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Ports       []Port         `json:"ports,omitempty"`
}

type JVB struct {
	Exporter Exporter `json:"exporter,omitempty"`
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
	//+kubebuilder:default="jitsi/jvb:stable-5390-3"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy    v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets   []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	Environments       []v1.EnvVar               `json:"environments,omitempty"`
	Resources          v1.ResourceRequirements   `json:"resources,omitempty"`
	ServiceAnnotations map[string]string         `json:"service_annotations,omitempty"`
	//+kubebuilder:default:="ClusterIP"
	ServiceType v1.ServiceType `json:"service_type,omitempty"`
	Port        Port           `json:"port,omitempty"`
}

type Exporter struct {
	Type          string `json:"type,omitempty"`
	ConfigMapName string `json:"config_map_name,omitempty"`
	//+kubebuilder:default:="systemli/prometheus-jitsi-meet-exporter:latest"
	Image string `json:"image,omitempty"`
	//+kubebuilder:default:=9888
	Port int32 `json:"port,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy  v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	Environments     []v1.EnvVar               `json:"environments,omitempty"`
	Resources        v1.ResourceRequirements   `json:"resources,omitempty"`
	SecurityContext  v1.SecurityContext        `json:"security_context,omitempty"`
}

type StorageSpec struct {
	EmptyDir *v1.EmptyDirVolumeSource `json:"empty_dir,omitempty"`
	PVC      v1.PersistentVolumeClaim `json:"pvc,omitempty"`
}

type Port struct {
	Protocol v1.Protocol `json:"protocol,omitempty"`
	//+kubebuilder:default="http"
	PortName string `json:"port_name,omitempty"`
	Port     int32  `json:"port,omitempty"`
}

// JitsiStatus defines the observed state of Jitsi
type JitsiStatus struct{}

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
