package manifests

import (
	"context"
	"github.com/onmetal/meeting-operator/apis/etherpad/v1alpha1"
	v1alpha12 "github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AppKubernetesPartOf    = "jitsi-meet"
	EtherpadDeploymentName = "etherpad"
	JicofoDeploymentName   = "jitsi-jicofo"
	JicofoContainerName    = "jicofo"
	ProsodyDeploymentName  = "jitsi-prosody"
	ProsodyContainerName   = "prosody"
	WebDeploymentName      = "jitsi-web"
	WebContainerName       = "web"
)

type Helper interface {
	Make() error
	Get(name, namespace string) (*interface{}, error)
	Create() error
	Update() error
	Delete() error
}

type DeploymentTemplate struct {
	Name, Namespace, Image, ContainerName string
	Replicas                              int32
	PortSpec                              map[string]int32
	ImagePullPolicy                       v1.PullPolicy
	Environments                          []v1.EnvVar
	Ctx                                   context.Context
	Labels                                map[string]string
	Client                                client.Client
	Log                                   logr.Logger
}

func (d *DeploymentTemplate) Make() error {
	_, err := d.Get()
	if err != nil && errors.IsNotFound(err) {
		return d.Create()
	}
	if err := d.Update(); err != nil {
		return err
	}
	d.Log.Info("deployment updated:", "Name", d.Name, "Namespace", d.Namespace)
	return nil
}

func (d *DeploymentTemplate) Create() error {
	preparedDeployment := d.prepare()
	return d.Client.Create(d.Ctx, preparedDeployment)
}

func (d *DeploymentTemplate) Update() error {
	deployment, err := d.Get()
	if err != nil {
		return err
	}
	updatedSpec := d.prepareSpec()
	deployment.Spec = updatedSpec
	return d.Client.Update(d.Ctx, deployment)
}

func (d *DeploymentTemplate) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: d.Name, Namespace: d.Namespace}, deployment)
	return deployment, err
}

func (d *DeploymentTemplate) Delete() error {
	deployment, err := d.Get()
	if err != nil {
		return err
	}
	return d.Client.Delete(d.Ctx, deployment)
}

func (d *DeploymentTemplate) prepare() *appsv1.Deployment {
	spec := d.prepareSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: d.Namespace,
			Labels:    d.Labels,
		},
		Spec: spec,
	}
}

func (d *DeploymentTemplate) prepareSpec() appsv1.DeploymentSpec {
	defaultLabels := GetDefaultLabels(d.Name)
	ports := d.getContainerPorts()
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: defaultLabels,
		},
		Replicas: &d.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: defaultLabels,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            d.ContainerName,
						Image:           d.Image,
						ImagePullPolicy: d.ImagePullPolicy,
						Env:             d.Environments,
						Ports:           ports,
					},
				},
			},
		},
	}
}
func (d *DeploymentTemplate) getContainerPorts() []v1.ContainerPort {
	var ports []v1.ContainerPort
	for name, port := range d.PortSpec {
		ports = append(ports, v1.ContainerPort{
			Name:          name,
			ContainerPort: port,
			Protocol:      "TCP",
		})
	}
	return ports
}

func GetDefaultLabels(appName string) map[string]string {
	var defaultLabels = make(map[string]string)

	defaultLabels["app.kubernetes.io/appName"] = appName
	defaultLabels["app.kubernetes.io/part-of"] = AppKubernetesPartOf

	return defaultLabels
}

func NewJitsiTemplate(ctx context.Context, appName string,
	j *v1alpha12.Jitsi, c client.Client, l logr.Logger) *DeploymentTemplate {
	switch appName {
	case WebContainerName:
		return &DeploymentTemplate{
			Name:            WebDeploymentName,
			Namespace:       j.Namespace,
			Image:           j.Spec.Web.Image,
			ContainerName:   WebContainerName,
			Replicas:        j.Spec.Web.Replicas,
			PortSpec:        j.Spec.Web.PortSpec,
			ImagePullPolicy: j.Spec.Web.ImagePullPolicy,
			Environments:    j.Spec.Web.Environments,
			Ctx:             ctx,
			Client:          c,
			Log:             l,
		}
	case ProsodyContainerName:
		return &DeploymentTemplate{
			Name:            ProsodyDeploymentName,
			Namespace:       j.Namespace,
			Image:           j.Spec.Prosody.Image,
			ContainerName:   ProsodyContainerName,
			Replicas:        j.Spec.Prosody.Replicas,
			PortSpec:        j.Spec.Prosody.PortSpec,
			ImagePullPolicy: j.Spec.Prosody.ImagePullPolicy,
			Environments:    j.Spec.Prosody.Environments,
			Ctx:             ctx,
			Client:          c,
			Log:             l,
		}
	case JicofoContainerName:
		return &DeploymentTemplate{
			Name:            JicofoDeploymentName,
			Namespace:       j.Namespace,
			Image:           j.Spec.Jicofo.Image,
			ContainerName:   JicofoContainerName,
			Replicas:        j.Spec.Jicofo.Replicas,
			PortSpec:        j.Spec.Jicofo.PortSpec,
			ImagePullPolicy: j.Spec.Jicofo.ImagePullPolicy,
			Environments:    j.Spec.Jicofo.Environments,
			Ctx:             ctx,
			Client:          c,
			Log:             l,
		}
	default:
		return &DeploymentTemplate{}
	}
}

func NewEtherpadTemplate(ctx context.Context, e *v1alpha1.Etherpad, c client.Client, l logr.Logger) *DeploymentTemplate {
	return &DeploymentTemplate{
		Name:            EtherpadDeploymentName,
		Namespace:       e.Namespace,
		Image:           e.Spec.Image,
		ContainerName:   EtherpadDeploymentName,
		Replicas:        e.Spec.Replicas,
		PortSpec:        e.Spec.PortSpec,
		ImagePullPolicy: e.Spec.ImagePullPolicy,
		Environments:    e.Spec.Environments,
		Ctx:             ctx,
		Client:          c,
		Log:             l,
	}
}
