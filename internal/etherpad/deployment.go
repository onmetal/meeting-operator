// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package etherpad

import (
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ Etherpad = (*etherpad)(nil)

const (
	applicationName = "etherpad"
)

func (e *etherpad) Create() error {
	preparedDeployment := e.prepareDeployment()
	svc := newService(e)
	if svcErr := svc.Create(); svcErr != nil {
		return svcErr
	}
	return e.Client.Create(e.ctx, preparedDeployment)
}

func (e *etherpad) prepareDeployment() *appsv1.Deployment {
	spec := e.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Name,
			Namespace: e.Namespace,
			Labels:    e.labels,
		},
		Spec: spec,
	}
}

func (e *etherpad) prepareDeploymentSpec() appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: e.labels,
		},
		Replicas: &e.Spec.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: e.labels,
			},
			Spec: v1.PodSpec{
				ImagePullSecrets: e.Spec.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            applicationName,
						Image:           e.Spec.Image,
						ImagePullPolicy: e.Spec.ImagePullPolicy,
						Env:             e.Spec.Environments,
						Ports:           e.getContainerPorts(),
						SecurityContext: &e.Spec.SecurityContext,
					},
				},
			},
		},
	}
}

func (e *etherpad) getContainerPorts() []v1.ContainerPort {
	containerPorts := make([]v1.ContainerPort, 0, len(e.Spec.Ports))
	for port := range e.Spec.Ports {
		containerPorts = append(containerPorts, v1.ContainerPort{
			Name:          e.Spec.Ports[port].Name,
			ContainerPort: e.Spec.Ports[port].Port,
			Protocol:      e.Spec.Ports[port].Protocol,
		})
	}
	return containerPorts
}

func (e *etherpad) Update() error {
	updatedDeployment := e.prepareDeployment()
	svc := newService(e)
	if svcErr := svc.Update(); svcErr != nil {
		return svcErr
	}
	return e.Client.Update(e.ctx, updatedDeployment)
}

func (e *etherpad) Delete() error {
	if err := utils.RemoveFinalizer(e.ctx, e.Client, e.Etherpad); err != nil {
		e.log.Info("can't remove finalizer", "error", err)
	}
	deployment, err := e.Get()
	if err != nil {
		return err
	}
	svc := newService(e)
	if svcErr := svc.Delete(); svcErr != nil {
		return svcErr
	}
	return e.Client.Delete(e.ctx, deployment)
}

func (e *etherpad) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := e.Client.Get(e.ctx, types.NamespacedName{
		Namespace: e.Namespace,
		Name:      e.Name,
	}, deployment)
	return deployment, err
}
