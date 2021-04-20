package jitsi

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (j *Jicofo) Create() error {
	preparedDeployment := j.prepareDeployment()
	return j.Client.Create(j.ctx, preparedDeployment)
}

func (j *Jicofo) prepareDeployment() *appsv1.Deployment {
	spec := j.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.name,
			Namespace: j.namespace,
			Labels:    j.labels,
		},
		Spec: spec,
	}
}

func (j *Jicofo) prepareDeploymentSpec() appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: j.labels,
		},
		Replicas: &j.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: j.labels,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            JicofoName,
						Image:           j.Image,
						ImagePullPolicy: j.ImagePullPolicy,
						Env:             j.Environments,
						Ports:           j.getContainerPorts(),
					},
				},
			},
		},
	}
}

func (j *Jicofo) getContainerPorts() []v1.ContainerPort {
	var ports []v1.ContainerPort
	for svc := range j.Services {
		ports = append(ports, v1.ContainerPort{
			Name:          j.Services[svc].PortName,
			ContainerPort: j.Services[svc].Port,
			Protocol:      j.Services[svc].Protocol,
		})
	}
	return ports
}

func (j *Jicofo) Update() error {
	updatedDeployment := j.prepareDeployment()
	return j.Client.Update(j.ctx, updatedDeployment)
}

func (j *Jicofo) Delete() error {
	deployment, err := j.Get()
	if err != nil {
		return err
	}
	return j.Client.Delete(j.ctx, deployment)
}

func (j *Jicofo) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := j.Client.Get(j.ctx, types.NamespacedName{
		Namespace: j.namespace,
		Name:      j.name,
	}, deployment)
	return deployment, err
}
