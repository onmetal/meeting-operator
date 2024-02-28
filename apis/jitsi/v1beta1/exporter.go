// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
