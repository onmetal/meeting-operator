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
	var ports []v1.ContainerPort
	for port := range e.Spec.Ports {
		ports = append(ports, v1.ContainerPort{
			Name:          e.Spec.Ports[port].Name,
			ContainerPort: e.Spec.Ports[port].Port,
			Protocol:      e.Spec.Ports[port].Protocol,
		})
	}
	return ports
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
	if utils.ContainsString(e.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		e.ObjectMeta.Finalizers = utils.RemoveString(e.ObjectMeta.Finalizers, utils.MeetingFinalizer)
		if err := e.Client.Update(e.ctx, e.Etherpad); err != nil {
			e.log.Info("can't update etherpad cr", "error", err)
		}
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
