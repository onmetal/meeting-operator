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
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/onmetal/meeting-operator/internal/utils"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WebName     = "web"
	ProsodyName = "prosody"
	JicofoName  = "jicofo"
	JibriName   = "jibri"
)

type Jitsi interface {
	Create() error
	Update() error
	Delete() error
}

type Web struct {
	client.Client
	*v1alpha1.Web

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

type Prosody struct {
	client.Client
	*v1alpha1.Prosody

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

type Jicofo struct {
	client.Client
	*v1alpha1.Jicofo

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

type Jibri struct {
	client.Client
	*v1alpha1.Jibri

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

type JVB struct {
	client.Client
	v1alpha1.JVB

	ctx                          context.Context
	log                          logr.Logger
	envs                         []v1.EnvVar
	name, serviceName, namespace string
	replica                      int32
}

func New(ctx context.Context, appName string,
	j *v1alpha1.Jitsi, c client.Client, l logr.Logger) (Jitsi, error) {
	switch appName {
	case WebName:
		labels := utils.GetDefaultLabels(WebName)
		return &Web{
			Client:    c,
			Web:       &j.Spec.Web,
			name:      WebName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    labels,
		}, nil
	case ProsodyName:
		labels := utils.GetDefaultLabels(ProsodyName)
		return &Prosody{
			Client:    c,
			Prosody:   &j.Spec.Prosody,
			name:      ProsodyName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    labels,
		}, nil
	case JicofoName:
		labels := utils.GetDefaultLabels(JicofoName)
		return &Jicofo{
			Client:    c,
			Jicofo:    &j.Spec.Jicofo,
			name:      JicofoName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    labels,
		}, nil
	case JibriName:
		labels := utils.GetDefaultLabels(JibriName)
		return &Jibri{
			Client:    c,
			Jibri:     &j.Spec.Jibri,
			name:      JibriName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    labels,
		}, nil
	default:
		return nil, fmt.Errorf(fmt.Sprintf("component: %s not exist", appName))
	}
}

func getContainerPorts(ports []v1alpha1.Port) []v1.ContainerPort {
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
