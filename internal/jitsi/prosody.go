package jitsi

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (p *Prosody) Create() error {
	preparedDeployment := p.prepareDeployment()
	return p.Client.Create(p.ctx, preparedDeployment)
}

func (p *Prosody) prepareDeployment() *appsv1.Deployment {
	spec := p.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.name,
			Namespace: p.namespace,
			Labels:    p.labels,
		},
		Spec: spec,
	}
}

func (p *Prosody) prepareDeploymentSpec() appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: p.labels,
		},
		Replicas: &p.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: p.labels,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            JicofoName,
						Image:           p.Image,
						ImagePullPolicy: p.ImagePullPolicy,
						Env:             p.Environments,
						Ports:           p.getContainerPorts(),
					},
				},
			},
		},
	}
}

func (p *Prosody) getContainerPorts() []v1.ContainerPort {
	var ports []v1.ContainerPort
	for svc := range p.Services {
		ports = append(ports, v1.ContainerPort{
			Name:          p.Services[svc].PortName,
			ContainerPort: p.Services[svc].Port,
			Protocol:      p.Services[svc].Protocol,
		})
	}
	return ports
}

func (p *Prosody) Update() error {
	updatedDeployment := p.prepareDeployment()
	return p.Client.Update(p.ctx, updatedDeployment)
}

func (p *Prosody) Delete() error {
	deployment, err := p.Get()
	if err != nil {
		return err
	}
	return p.Client.Delete(p.ctx, deployment)
}

func (p *Prosody) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := p.Client.Get(p.ctx, types.NamespacedName{
		Namespace: p.namespace,
		Name:      p.name,
	}, deployment)
	return deployment, err
}
