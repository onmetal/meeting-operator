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
						Ports:           getContainerPorts(w.Services),
						Resources:       w.Resources,
					},
				},
			},
		},
	}
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
