package jitsi

import (
	"context"

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
	services        []v1alpha1.Service
	serviceType     v1.ServiceType
	labels          map[string]string
}

func NewService(ctx context.Context, appName string,
	j *v1alpha1.Jitsi, c client.Client, l logr.Logger) Jitsi {
	switch appName {
	case WebName:
		labels := getDefaultLabels(WebName)
		return &Service{
			Client:      c,
			services:    j.Spec.Web.Services,
			serviceType: j.Spec.Web.ServiceType,
			name:        WebName,
			namespace:   j.Namespace,
			ctx:         ctx,
			log:         l,
			labels:      labels,
		}
	case ProsodyName:
		labels := getDefaultLabels(ProsodyName)
		return &Service{
			Client:      c,
			services:    j.Spec.Prosody.Services,
			serviceType: j.Spec.Prosody.ServiceType,
			name:        ProsodyName,
			namespace:   j.Namespace,
			ctx:         ctx,
			log:         l,
			labels:      labels,
		}
	case JicofoName:
		labels := getDefaultLabels(JicofoName)
		return &Service{
			Client:      c,
			services:    j.Spec.Jicofo.Services,
			serviceType: j.Spec.Jicofo.ServiceType,
			name:        JicofoName,
			namespace:   j.Namespace,
			ctx:         ctx,
			log:         l,
			labels:      labels,
		}
	default:
		return nil
	}
}
func (s *Service) Create() error {
	preparedService := s.prepareService()
	return s.Client.Create(s.ctx, preparedService)
}

func (s *Service) prepareService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: s.namespace,
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
	case service.Spec.Type != s.serviceType:
		// You can't change spec.type on existing service
		if err := s.Client.Delete(s.ctx, service); err != nil {
			s.log.Error(err, "failed to delete ServiceTemplate")
		}
		preparedService := s.prepareService()
		return s.Client.Create(s.ctx, preparedService)
	default:
		updatedServiceSpec := s.prepareServiceSpec()
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
