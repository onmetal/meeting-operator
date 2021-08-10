package v1beta1

type Exporter struct {
	DeploymentSpec `json:"deployment_spec"`
	Type          string `json:"type,omitempty"`
	ConfigMapName string `json:"config_map_name,omitempty"`
	//+kubebuilder:default:=9888
	Port int32 `json:"port,omitempty"`
}
