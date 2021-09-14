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
