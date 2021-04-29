package jitsi

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (w *Web) Create() error {
	preparedDeployment := w.prepareDeployment()
	return w.Client.Create(w.ctx, preparedDeployment)
}

func (w *Web) prepareDeployment() *appsv1.Deployment {
	spec := w.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.name,
			Namespace: w.namespace,
			Labels:    w.labels,
		},
		Spec: spec,
	}
}

func (w *Web) prepareDeploymentSpec() appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: w.labels,
		},
		Replicas: &w.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: w.labels,
			},
			Spec: v1.PodSpec{
				ImagePullSecrets: w.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            WebName,
						Image:           w.Image,
						ImagePullPolicy: w.ImagePullPolicy,
						Env:             w.Environments,
						Ports:           w.getContainerPorts(),
					},
				},
			},
		},
	}
}

func (w *Web) getContainerPorts() []v1.ContainerPort {
	var ports []v1.ContainerPort
	for svc := range w.Services {
		ports = append(ports, v1.ContainerPort{
			Name:          w.Services[svc].PortName,
			ContainerPort: w.Services[svc].Port,
			Protocol:      w.Services[svc].Protocol,
		})
	}
	return ports
}

func (w *Web) Update() error {
	updatedDeployment := w.prepareDeployment()
	return w.Client.Update(w.ctx, updatedDeployment)
}

func (w *Web) Delete() error {
	deployment, err := w.Get()
	if err != nil {
		return err
	}
	return w.Client.Delete(w.ctx, deployment)
}

func (w *Web) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := w.Client.Get(w.ctx, types.NamespacedName{
		Namespace: w.namespace,
		Name:      w.name,
	}, deployment)
	return deployment, err
}
