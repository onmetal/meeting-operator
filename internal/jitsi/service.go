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

	"github.com/onmetal/meeting-operator/internal/utils"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Service struct {
	client.Client

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	serviceType     v1.ServiceType
	services        []v1alpha1.Service
	annotations     map[string]string
	labels          map[string]string
}

func NewService(ctx context.Context, appName string,
	j *v1alpha1.Jitsi, c client.Client, l logr.Logger) Jitsi {
	switch appName {
	case WebName:
		labels := utils.GetDefaultLabels(WebName)
		return &Service{
			Client:      c,
			services:    j.Spec.Web.Services,
			serviceType: j.Spec.Web.ServiceType,
			name:        WebName,
			namespace:   j.Namespace,
			ctx:         ctx,
			log:         l,
			annotations: j.Spec.Web.ServiceAnnotations,
			labels:      labels,
		}
	case ProsodyName:
		labels := utils.GetDefaultLabels(ProsodyName)
		return &Service{
			Client:      c,
			services:    j.Spec.Prosody.Services,
			serviceType: j.Spec.Prosody.ServiceType,
			name:        ProsodyName,
			namespace:   j.Namespace,
			ctx:         ctx,
			log:         l,
			annotations: j.Spec.Prosody.ServiceAnnotations,
			labels:      labels,
		}
	case JicofoName:
		labels := utils.GetDefaultLabels(JicofoName)
		return &Service{
			Client:      c,
			services:    j.Spec.Jicofo.Services,
			serviceType: j.Spec.Jicofo.ServiceType,
			name:        JicofoName,
			namespace:   j.Namespace,
			ctx:         ctx,
			log:         l,
			annotations: j.Spec.Jicofo.ServiceAnnotations,
			labels:      labels,
		}
	case JibriName:
		labels := utils.GetDefaultLabels(JibriName)
		return &Service{
			Client:      c,
			services:    j.Spec.Jibri.Services,
			serviceType: j.Spec.Jibri.ServiceType,
			name:        JibriName,
			namespace:   j.Namespace,
			ctx:         ctx,
			log:         l,
			annotations: j.Spec.Jibri.ServiceAnnotations,
			labels:      labels,
		}
	default:
		return &Service{}
	}
}
func (s *Service) Create() error {
	preparedService := s.prepareService()
	return s.Client.Create(s.ctx, preparedService)
}

func (s *Service) prepareService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        s.name,
			Namespace:   s.namespace,
			Annotations: s.annotations,
		},
		Spec: s.prepareServiceSpec(),
	}
}

func (s *Service) prepareServiceSpec() v1.ServiceSpec {
	return v1.ServiceSpec{
		Type:     s.serviceType,
		Ports:    getPorts(s.services),
		Selector: s.labels,
	}
}

func (s *Service) Update() error {
	service, err := s.Get()
	if err != nil {
		return err
	}
	switch {
	case service.Spec.Type == "LoadBalancer":

		// can't delete annotations when service type is LoadBalancer
		if compareServiceAnnotations(service.Annotations, s.annotations) {
			if err := s.Client.Delete(s.ctx, service); err != nil {
				s.log.Error(err, "failed to delete service")
			}
			preparedService := s.prepareService()
			return s.Client.Create(s.ctx, preparedService)
		}
		updatedServiceSpec := s.prepareServiceSpec()
		service.Annotations = s.annotations
		service.Spec.Ports = updatedServiceSpec.Ports
		service.Spec.Selector = updatedServiceSpec.Selector
		return s.Client.Update(s.ctx, service)
	case service.Spec.Type != s.serviceType:
		// You can't change spec.type on existing service
		if err := s.Client.Delete(s.ctx, service); err != nil {
			s.log.Error(err, "failed to delete service")
		}
		preparedService := s.prepareService()
		return s.Client.Create(s.ctx, preparedService)
	default:
		updatedServiceSpec := s.prepareServiceSpec()
		service.Annotations = s.annotations
		service.Spec.Ports = updatedServiceSpec.Ports
		service.Spec.Selector = updatedServiceSpec.Selector
		return s.Client.Update(s.ctx, service)
	}
}

func (s *Service) Get() (*v1.Service, error) {
	service := &v1.Service{}
	err := s.Client.Get(s.ctx, types.NamespacedName{
		Name:      s.name,
		Namespace: s.namespace,
	}, service)
	return service, err
}

func (s *Service) Delete() error {
	service, err := s.Get()
	if err != nil {
		if errors.IsNotFound(err) {
			s.log.Info("service not found", "Name", s.name)
			return nil
		}
		s.log.Info("failed to get service", "error", err)
		return nil
	}
	return s.Client.Delete(s.ctx, service)
}

func compareServiceAnnotations(oldAnnotations, newAnnotations map[string]string) bool {
	if len(newAnnotations) < 1 {
		return true
	}
	for name := range oldAnnotations {
		if _, ok := newAnnotations[name]; !ok {
			return true
		}
	}
	return false
}

func getPorts(services []v1alpha1.Service) []v1.ServicePort {
	p := make([]v1.ServicePort, 0, 1)
	for svc := range services {
		p = append(p, v1.ServicePort{
			Name:       services[svc].PortName,
			TargetPort: intstr.IntOrString{IntVal: services[svc].Port},
			Port:       services[svc].Port,
			Protocol:   services[svc].Protocol,
		})
	}
	return p
}
