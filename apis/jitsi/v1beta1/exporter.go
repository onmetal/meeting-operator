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

type Exporter struct {
	Type string `json:"type,omitempty"`
	//+kubebuilder:default="systemli/prometheus-jitsi-meet-exporter:latest"
	Image           string                  `json:"image,omitempty"`
	ConfigMapName   string                  `json:"config_map_name,omitempty"`
	SecurityContext v1.SecurityContext      `json:"security_context,omitempty"`
	Environments    []v1.EnvVar             `json:"environments,omitempty"`
	Resources       v1.ResourceRequirements `json:"resources,omitempty"`
	//+kubebuilder:default:=9888
	Port int32 `json:"port,omitempty"`
}
