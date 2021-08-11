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

package jitsi

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WebName     = "web"
	ProsodyName = "prosody"
	JicofoName  = "jicofo"
	JibriName   = "jibri"
	JigasiName  = "jigasi"
)

type Jitsi interface {
	Create() error
	Update() error
	Delete() error
}







//type Jibri struct {
//	client.Client
//	*v1alpha1.Jibri
//
//	ctx             context.Context
//	log             logr.Logger
//	name, namespace string
//	labels          map[string]string
//}

//type Jigasi struct {
//	client.Client
//	*v1alpha1.Jigasi
//
//	ctx             context.Context
//	log             logr.Logger
//	name, namespace string
//	labels          map[string]string
//}

type JVB struct {
	client.Client
	*v1beta1.JVB

	ctx                          context.Context
	log                          logr.Logger
	envs                         []v1.EnvVar
	name, serviceName, namespace string
	replica                      int32
	deleted                      bool
}



func getContainerPorts(ports []v1beta1.Port) []v1.ContainerPort {
	var containerPorts []v1.ContainerPort
	if len(ports) < 1 {
		return nil
	}
	for svc := range ports {
		containerPorts = append(containerPorts, v1.ContainerPort{
			Name:          ports[svc].Name,
			ContainerPort: ports[svc].Port,
			Protocol:      ports[svc].Protocol,
		})
	}
	return containerPorts
}
