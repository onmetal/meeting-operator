package manifests

import (
	"context"

	"github.com/go-logr/logr"
	jitsiv1alpha1 "github.com/onmetal/meeting-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	JicofoServiceName  = "jitsi-jicofo"
	ProsodyServiceName = "jitsi-prosody"
	WebServiceName     = "jitsi-web"
)

type ServiceTemplate struct {
	Name, SelectorName, Namespace string
	ServiceType                   v1.ServiceType
	PortSpec                      map[string]int32
	Ctx                           context.Context
	Client                        client.Client
	Log                           logr.Logger
}

func (s *ServiceTemplate) Make() error {
	_, err := s.Get()
	if err != nil && errors.IsNotFound(err) {
		return s.Create()
	}
	return s.Update()
}

func (s *ServiceTemplate) Create() error {
	preparedService := s.prepareService()
	return s.Client.Create(s.Ctx, preparedService)
}

func (s *ServiceTemplate) Update() error {
	service, err := s.Get()
	if err != nil {
		return err
	}
	switch {
	case service.Spec.Type != s.ServiceType:
		// You can's change spec.type on existing ServiceTemplate
		if err := s.Client.Delete(s.Ctx, service); err != nil {
			s.Log.Error(err, "failed to delete ServiceTemplate")
		}
		preparedService := s.prepareService()
		return s.Client.Create(s.Ctx, preparedService)
	default:
		if err := s.Client.Delete(s.Ctx, service); err != nil {
			s.Log.Error(err, "failed to delete ServiceTemplate")
		}
		preparedService := s.prepareService()
		return s.Client.Create(s.Ctx, preparedService)
	}
}
func (s *ServiceTemplate) Get() (*v1.Service, error) {
	service := &v1.Service{}
	err := s.Client.Get(s.Ctx, types.NamespacedName{Name: s.Name, Namespace: s.Namespace}, service)
	return service, err
}

func (s *ServiceTemplate) Delete() error {
	service, err := s.Get()
	if err != nil {
		if errors.IsNotFound(err) {
			s.Log.Info("ServiceTemplate not found", "Name", s.Name)
			return nil
		}
		s.Log.Info("failed to get ServiceTemplate", "error", err)
		return nil
	}
	return s.Client.Delete(s.Ctx, service)
}

func (s *ServiceTemplate) prepareService() *v1.Service {
	spec := s.prepareServiceSpec()
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Spec: spec,
	}
}

func (s *ServiceTemplate) prepareServiceSpec() v1.ServiceSpec {
	selectorLabels := setLabelsForApp(s.SelectorName)
	ports := getPorts(s.PortSpec)
	return v1.ServiceSpec{
		Type:     s.ServiceType,
		Ports:    ports,
		Selector: selectorLabels,
	}
}
func getPorts(ports map[string]int32) []v1.ServicePort {
	var p []v1.ServicePort

	for name, port := range ports {
		p = append(p, v1.ServicePort{
			Name:       name,
			TargetPort: intstr.IntOrString{IntVal: port},
			Port:       port,
			Protocol:   "TCP",
		})
	}
	return p
}

func setLabelsForApp(appName string) map[string]string {
	var labels = make(map[string]string)

	labels["app.kubernetes.io/appName"] = appName
	labels["app.kubernetes.io/part-of"] = AppKubernetesPartOf

	return labels
}

func NewJitsiServiceTemplate(ctx context.Context, appName string, j *jitsiv1alpha1.Jitsi,
	c client.Client, l logr.Logger) *ServiceTemplate {
	switch appName {
	case "web":
		return &ServiceTemplate{
			Name:         WebServiceName,
			SelectorName: WebDeploymentName,
			Namespace:    j.Namespace,
			ServiceType:  j.Spec.Web.Type,
			PortSpec:     j.Spec.Web.PortSpec,
			Ctx:          ctx,
			Client:       c,
			Log:          l,
		}
	case "prosody":
		return &ServiceTemplate{
			Name:         ProsodyServiceName,
			SelectorName: ProsodyDeploymentName,
			Namespace:    j.Namespace,
			ServiceType:  j.Spec.Prosody.Type,
			PortSpec:     j.Spec.Prosody.PortSpec,
			Ctx:          ctx,
			Client:       c,
			Log:          l,
		}
	case "jicofo":
		return &ServiceTemplate{
			Name:         JicofoServiceName,
			SelectorName: JicofoDeploymentName,
			Namespace:    j.Namespace,
			ServiceType:  j.Spec.Jicofo.Type,
			PortSpec:     j.Spec.Jicofo.PortSpec,
			Ctx:          ctx,
			Client:       c,
			Log:          l,
		}
	default:
		return &ServiceTemplate{}
	}
}

func NewEtherpadServiceTemplate(ctx context.Context, e *jitsiv1alpha1.Etherpad,
	c client.Client, l logr.Logger) *ServiceTemplate {
	return &ServiceTemplate{
		Name:         e.Name,
		SelectorName: EtherpadDeploymentName,
		Namespace:    e.Namespace,
		ServiceType:  e.Spec.Type,
		PortSpec:     e.Spec.PortSpec,
		Ctx:          ctx,
		Client:       c,
		Log:          l,
	}
}
