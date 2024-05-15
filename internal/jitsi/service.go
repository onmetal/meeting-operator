// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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

const (
	WebAppName     = "web"
	ProsodyAppName = "prosody"
	JibriAppName   = "jibri"
)

type Servicer interface {
	Create() error
	Update() error
	Delete() error
}

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

func NewService(ctx context.Context, c client.Client, l logr.Logger,
	appName, namespace string,
	annotations, labels map[string]string,
	serviceType v1.ServiceType, ports []v1beta1.Port,
) Servicer {
	switch appName {
	case WebAppName:
		return &service{
			Client:      c,
			ports:       getServicePortsForApp(appName, ports),
			serviceType: serviceType,
			name:        appName,
			namespace:   namespace,
			ctx:         ctx,
			log:         l,
			annotations: annotations,
			labels:      labels,
		}
	case ProsodyAppName:
		return &service{
			Client:      c,
			ports:       getServicePortsForApp(appName, ports),
			serviceType: serviceType,
			name:        appName,
			namespace:   namespace,
			ctx:         ctx,
			log:         l,
			annotations: annotations,
			labels:      labels,
		}
	case JibriAppName:
		return &service{
			Client:      c,
			ports:       getServicePortsForApp(appName, ports),
			serviceType: serviceType,
			name:        appName,
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
	err := s.Client.Create(s.ctx, preparedService)
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
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

const (
	defaultHTTPPort = 80
	xmppPort        = 5222
)

const (
	prosodyHTTPPort     = 5280
	prosodyC2SPort      = 5282
	prosodyExternalPort = 5347
)

func getServicePortsForApp(appName string, s []v1beta1.Port) []v1.ServicePort {
	switch appName {
	case WebAppName:
		ports := make([]v1.ServicePort, 0, 1) //nolint:mnd //reason: just minimal value
		ports = append(ports, v1.ServicePort{
			Name: "http", Protocol: v1.ProtocolTCP,
			Port: defaultHTTPPort, TargetPort: intstr.IntOrString{IntVal: defaultHTTPPort},
		})
		if len(s) != 0 {
			return getAdditionalPorts(ports, s)
		}
		return ports
	case ProsodyAppName:
		ports := make([]v1.ServicePort, 0, 4) //nolint:mnd //reason: just minimal value
		ports = append(ports,
			v1.ServicePort{
				Name: "http", Protocol: v1.ProtocolTCP,
				Port: prosodyHTTPPort, TargetPort: intstr.IntOrString{IntVal: prosodyHTTPPort},
			},
			v1.ServicePort{
				Name: "c2s", Protocol: v1.ProtocolTCP,
				Port: prosodyC2SPort, TargetPort: intstr.IntOrString{IntVal: prosodyC2SPort},
			},
			v1.ServicePort{
				Name: "xmpp", Protocol: v1.ProtocolTCP,
				Port: xmppPort, TargetPort: intstr.IntOrString{IntVal: xmppPort},
			},
			v1.ServicePort{
				Name: "external", Protocol: v1.ProtocolTCP,
				Port: prosodyExternalPort, TargetPort: intstr.IntOrString{IntVal: prosodyExternalPort},
			},
		)
		if len(s) != 0 {
			return getAdditionalPorts(ports, s)
		}
		return ports
	case JibriAppName:
		ports := make([]v1.ServicePort, 0, 1) //nolint:mnd //reason: just minimal value
		ports = append(ports, v1.ServicePort{
			Name: "http", Protocol: v1.ProtocolTCP,
			Port: xmppPort, TargetPort: intstr.IntOrString{IntVal: xmppPort},
		})
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
			Port: ports[port].Port, Protocol: ports[port].Protocol,
		})
	}
	return servicePorts
}

func GetContainerPorts(ports []v1beta1.Port) []v1.ContainerPort {
	if len(ports) == 0 {
		return nil
	}
	containerPorts := make([]v1.ContainerPort, 0, len(ports)) //nolint:mnd //reason: just minimal value
	for svc := range ports {
		containerPorts = append(containerPorts, v1.ContainerPort{
			Name:          ports[svc].Name,
			ContainerPort: ports[svc].Port,
			Protocol:      ports[svc].Protocol,
		})
	}
	return containerPorts
}
