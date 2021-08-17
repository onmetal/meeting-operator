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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type service struct {
	client.Client

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	annotations     map[string]string
	serviceType     v1.ServiceType
	ports           []v1.ServicePort
	labels          map[string]string
}

func newService(ctx context.Context, c client.Client, l logr.Logger,
	appName, namespace string,
	annotations, labels map[string]string,
	serviceType v1.ServiceType, ports []v1beta1.Port) *service {
	switch appName {
	case WebName:
		return &service{
			Client:      c,
			ports:       getPorts(WebName, ports),
			serviceType: serviceType,
			name:        WebName,
			namespace:   namespace,
			ctx:         ctx,
			log:         l,
			annotations: annotations,
			labels:      labels,
		}
	case ProsodyName:
		return &service{
			Client:      c,
			ports:       getPorts(ProsodyName, ports),
			serviceType: serviceType,
			name:        ProsodyName,
			namespace:   namespace,
			ctx:         ctx,
			log:         l,
			annotations: annotations,
			labels:      labels,
		}
	case JibriName:
		return &service{
			Client:      c,
			ports:       getPorts(JibriName, ports),
			serviceType: serviceType,
			name:        JibriName,
			namespace:   namespace,
			ctx:         ctx,
			log:         l,
			annotations: annotations,
			labels:      labels,
		}
	default:
		return nil
	}
}

func (s *service) Create() error {
	preparedService := s.prepareService()
	return s.Client.Create(s.ctx, preparedService)
}

func (s *service) prepareService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        s.name,
			Namespace:   s.namespace,
			Annotations: s.annotations,
		},
		Spec: s.prepareServiceSpec(),
	}
}

func (s *service) prepareServiceSpec() v1.ServiceSpec {
	return v1.ServiceSpec{
		Type:     s.serviceType,
		Ports:    s.ports,
		Selector: s.labels,
	}
}

func (s *service) Update() error {
	service, err := s.Get()
	if err != nil {
		return err
	}
	updatedServiceSpec := s.prepareServiceSpec()
	service.Annotations = s.annotations
	service.Spec.Type = s.serviceType
	service.Spec.Ports = updatedServiceSpec.Ports
	service.Spec.Selector = updatedServiceSpec.Selector
	return s.Client.Update(s.ctx, service)
}

func (s *service) Get() (*v1.Service, error) {
	service := &v1.Service{}
	err := s.Client.Get(s.ctx, types.NamespacedName{
		Name:      s.name,
		Namespace: s.namespace,
	}, service)
	return service, err
}

func (s *service) Delete() error {
	service, err := s.Get()
	if err != nil {
		if apierrors.IsNotFound(err) {
			s.log.Info("service not found", "Name", s.name)
			return nil
		}
		s.log.Info("failed to get service", "error", err)
		return nil
	}
	return s.Client.Delete(s.ctx, service)
}

func getPorts(appName string, s []v1beta1.Port) []v1.ServicePort {
	switch appName {
	case WebName:
		ports := make([]v1.ServicePort, 0, 1)
		ports = append(ports, v1.ServicePort{
			Name: "http", Protocol: v1.ProtocolTCP,
			Port: 80, TargetPort: intstr.IntOrString{IntVal: 80}})
		if len(s) != 0 {
			return getAdditionalPorts(ports, s)
		}
		return ports
	case ProsodyName:
		ports := make([]v1.ServicePort, 0, 4)
		ports = append(ports,
			v1.ServicePort{
				Name: "http", Protocol: v1.ProtocolTCP,
				Port: 5280, TargetPort: intstr.IntOrString{IntVal: 5280}},
			v1.ServicePort{
				Name: "c2s", Protocol: v1.ProtocolTCP,
				Port: 5282, TargetPort: intstr.IntOrString{IntVal: 5282}},
			v1.ServicePort{
				Name: "xmpp", Protocol: v1.ProtocolTCP,
				Port: 5222, TargetPort: intstr.IntOrString{IntVal: 5222}},
			v1.ServicePort{
				Name: "external", Protocol: v1.ProtocolTCP,
				Port: 5347, TargetPort: intstr.IntOrString{IntVal: 5347}},
		)
		if len(s) != 0 {
			return getAdditionalPorts(ports, s)
		}
		return ports
	case JibriName:
		ports := make([]v1.ServicePort, 0, 1)
		ports = append(ports, v1.ServicePort{
			Name: "http", Protocol: v1.ProtocolTCP,
			Port: 5282, TargetPort: intstr.IntOrString{IntVal: 5282}})
		if len(s) != 0 {
			return getAdditionalPorts(ports, s)
		}
		return ports
	default:
		return []v1.ServicePort{}
	}
}

func getAdditionalPorts(servicePorts []v1.ServicePort, ports []v1beta1.Port) []v1.ServicePort {
	for port := range ports {
		servicePorts = append(servicePorts, v1.ServicePort{
			Name: ports[port].Name, TargetPort: intstr.IntOrString{IntVal: ports[port].Port},
			Port: ports[port].Port, Protocol: ports[port].Protocol})
	}
	return servicePorts
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
