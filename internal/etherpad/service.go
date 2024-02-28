// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package etherpad

import (
	"github.com/onmetal/meeting-operator/apis/etherpad/v1alpha2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ Etherpad = (*service)(nil)

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
		Ports:    getPorts(s.ports),
		Selector: s.labels,
	}
}

func (s *service) Update() error {
	svc, err := s.Get()
	if err != nil {
		return err
	}
	updatedServiceSpec := s.prepareServiceSpec()
	svc.Spec.Ports = updatedServiceSpec.Ports
	svc.Spec.Selector = updatedServiceSpec.Selector
	return s.Client.Update(s.ctx, svc)
}

func (s *service) Get() (*v1.Service, error) {
	svc := &v1.Service{}
	err := s.Client.Get(s.ctx, types.NamespacedName{
		Namespace: s.namespace,
		Name:      s.name,
	}, svc)
	return svc, err
}

func (s *service) Delete() error {
	svc, err := s.Get()
	if err != nil {
		if errors.IsNotFound(err) {
			s.log.Info("service not found", "Name", s.name)
			return nil
		}
		s.log.Info("can't to get service", "error", err)
		return nil
	}
	return s.Client.Delete(s.ctx, svc)
}

func getPorts(ports []v1alpha2.Port) []v1.ServicePort {
	p := make([]v1.ServicePort, 0, 1)
	for svc := range ports {
		p = append(p, v1.ServicePort{
			Name:       ports[svc].Name,
			TargetPort: intstr.IntOrString{IntVal: ports[svc].Port},
			Port:       ports[svc].Port,
			Protocol:   ports[svc].Protocol,
		})
	}
	return p
}
