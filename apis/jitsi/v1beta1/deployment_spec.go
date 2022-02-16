// /*
// Copyright (c) 2021 T-Systems International GmbH, SAP SE or an SAP affiliate company. All right reserved
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package v1beta1

import v1 "k8s.io/api/core/v1"

type DeploymentSpec struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	//+kubebuilder:default:=1
	Replicas int32  `json:"replicas,omitempty"`
	Image    string `json:"image"`
	//+kubebuilder:default:=60
	TerminationGracePeriodSeconds int64 `json:"terminationGracePeriodSeconds,omitempty"`
	//+kubebuilder:default="IfNotPresent"
	ImagePullPolicy  v1.PullPolicy             `json:"image_pull_policy,omitempty"`
	ImagePullSecrets []v1.LocalObjectReference `json:"image_pull_secrets,omitempty"`
	SecurityContext  v1.SecurityContext        `json:"security_context,omitempty"`
	Environments     []v1.EnvVar               `json:"environments,omitempty"`
	Resources        v1.ResourceRequirements   `json:"resources,omitempty"`
}
