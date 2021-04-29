package etherpad

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ Etherpad = (*Deployment)(nil)

const (
	applicationName = "etherpad"
)

func (d *Deployment) Create() error {
	preparedDeployment := d.prepareDeployment()
	return d.Client.Create(d.ctx, preparedDeployment)
}

func (d *Deployment) prepareDeployment() *appsv1.Deployment {
	defaultLabels := getDefaultLabels()
	spec := d.prepareDeploymentSpec(defaultLabels)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.e.Name,
			Namespace: d.e.Namespace,
			Labels:    getDefaultLabels(),
		},
		Spec: spec,
	}
}

func (d *Deployment) prepareDeploymentSpec(defaultLabels map[string]string) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: defaultLabels,
		},
		Replicas: &d.e.Spec.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: defaultLabels,
			},
			Spec: v1.PodSpec{
				ImagePullSecrets: d.e.Spec.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            applicationName,
						Image:           d.e.Spec.Image,
						ImagePullPolicy: d.e.Spec.ImagePullPolicy,
						Env:             d.e.Spec.Environments,
						Ports:           d.getContainerPorts(),
					},
				},
			},
		},
	}
}

func (d *Deployment) getContainerPorts() []v1.ContainerPort {
	var ports []v1.ContainerPort
	for svc := range d.e.Spec.Services {
		ports = append(ports, v1.ContainerPort{
			Name:          d.e.Spec.Services[svc].PortName,
			ContainerPort: d.e.Spec.Services[svc].Port,
			Protocol:      d.e.Spec.Services[svc].Protocol,
		})
	}
	return ports
}

func (d *Deployment) Update() error {
	updatedDeployment := d.prepareDeployment()
	return d.Client.Update(d.ctx, updatedDeployment)
}

func (d *Deployment) Delete() error {
	deployment, err := d.Get()
	if err != nil {
		return err
	}
	return d.Client.Delete(d.ctx, deployment)
}

func (d *Deployment) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := d.Client.Get(d.ctx, types.NamespacedName{
		Namespace: d.e.Namespace,
		Name:      d.e.Name,
	}, deployment)
	return deployment, err
}
