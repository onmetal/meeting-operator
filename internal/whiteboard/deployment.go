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

package whiteboard

import (
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type WhiteBoard interface {
	Create() error
	Update() error
	Delete() error
}

func (w *whiteboard) Create() error {
	s := newService(w)
	if err := s.Create(); err != nil {
		return err
	}
	preparedDeployment := w.prepareDeployment()
	return w.Client.Create(w.ctx, preparedDeployment)
}

func (w *whiteboard) prepareDeployment() *appsv1.Deployment {
	spec := w.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        w.Name,
			Namespace:   w.Namespace,
			Labels:      w.labels,
			Annotations: w.Spec.Annotations,
		},
		Spec: spec,
	}
}

func (w *whiteboard) prepareDeploymentSpec() appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: w.labels,
		},
		Replicas: &w.Spec.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: w.labels,
			},
			Spec: v1.PodSpec{
				ImagePullSecrets: w.Spec.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            w.Name,
						Image:           w.Spec.Image,
						ImagePullPolicy: w.Spec.ImagePullPolicy,
						Env:             w.Spec.Environments,
						Ports:           w.getContainerPorts(),
						SecurityContext: &w.Spec.SecurityContext,
					},
				},
			},
		},
	}
}

func (w *whiteboard) getContainerPorts() []v1.ContainerPort {
	containerPorts := make([]v1.ContainerPort, 0, len(w.Spec.Ports))
	for p := range w.Spec.Ports {
		containerPorts = append(containerPorts, v1.ContainerPort{
			Name:          w.Spec.Ports[p].Name,
			ContainerPort: w.Spec.Ports[p].Port,
			Protocol:      w.Spec.Ports[p].Protocol,
		})
	}
	return containerPorts
}

func (w *whiteboard) Update() error {
	updatedDeployment := w.prepareDeployment()
	return w.Client.Update(w.ctx, updatedDeployment)
}

func (w *whiteboard) Delete() error {
	if err := utils.RemoveFinalizer(w.ctx, w.Client, w.WhiteBoard); err != nil {
		w.log.Info("can't remove finalizer", "error", err)
	}
	deployment, err := w.Get()
	if err != nil {
		return err
	}
	s := newService(w)
	if svcErr := s.Delete(); svcErr != nil {
		return svcErr
	}
	return w.Client.Delete(w.ctx, deployment)
}

func (w *whiteboard) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := w.Client.Get(w.ctx, types.NamespacedName{
		Namespace: w.Namespace,
		Name:      w.Name,
	}, deployment)
	return deployment, err
}
