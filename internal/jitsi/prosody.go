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

func (p *Prosody) Create() error {
	preparedDeployment := p.prepareDeployment()
	return p.Client.Create(p.ctx, preparedDeployment)
}

func (p *Prosody) prepareDeployment() *appsv1.Deployment {
	spec := p.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.name,
			Namespace:   p.namespace,
			Labels:      p.labels,
			Annotations: p.Annotations,
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
				ImagePullSecrets: p.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            JicofoName,
						Image:           p.Image,
						ImagePullPolicy: p.ImagePullPolicy,
						Env:             p.Environments,
						Ports:           getContainerPorts(p.Ports),
						Resources:       p.Resources,
						SecurityContext: &p.SecurityContext,
					},
				},
			},
		},
	}
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
