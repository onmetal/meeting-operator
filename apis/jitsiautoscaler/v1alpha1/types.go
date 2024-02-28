// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MetricName string

const (
	ResourceCPU          MetricName = "cpu"
	ResourceConference   MetricName = "jitsi_conference"
	ResourceParticipants MetricName = "jitsi_participants"
)

// AutoScalerSpec defines the desired state of AutoScaler.
type AutoScalerSpec struct {
	Labels         map[string]string `json:"labels,omitempty"`
	MonitoringType string            `json:"monitoringType,omitempty"`
	Host           string            `json:"host"`
	Interval       string            `json:"interval,omitempty"`
	Auth           Auth              `json:"auth,omitempty"`
	ScaleTargetRef ScaleTargetRef    `json:"scaleTargetRef,omitempty"`
	MinReplicas    int32             `json:"minReplicas,omitempty"`
	MaxReplicas    int32             `json:"maxReplicas,omitempty"`
	Metric         Metric            `json:"metric,omitempty"`
}

// ScaleTargetRef contains enough information to let you identify the referred resource.
type ScaleTargetRef struct {
	Name string `json:"name"`
}

type Metric struct {
	Name                     MetricName `json:"name"`
	TargetAverageUtilization int32      `json:"targetAverageUtilization"`
}

type Auth struct {
	Login    string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// AutoScalerStatus defines the observed state of AutoScaler.
type AutoScalerStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AutoScaler is the Schema for the autoScalers API.
type AutoScaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AutoScalerSpec   `json:"spec,omitempty"`
	Status AutoScalerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AutoScalerList contains a list of AutoScaler.
type AutoScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AutoScaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AutoScaler{}, &AutoScalerList{})
}
