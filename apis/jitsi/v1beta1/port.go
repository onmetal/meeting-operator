package v1beta1

import v1 "k8s.io/api/core/v1"

type Port struct {
	Name     string      `json:"name"`
	Protocol v1.Protocol `json:"protocol,omitempty"`
	Port     int32       `json:"port,omitempty"`
}
