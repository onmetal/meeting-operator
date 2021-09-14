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

package jigasi

import (
	"github.com/onmetal/meeting-operator/internal/jitsi"
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const name = "jigasi"

func (j *Jigasi) Create() error {
	preparedDeployment := j.prepareDeployment()
	return j.Client.Create(j.ctx, preparedDeployment)
}

func (j *Jigasi) prepareDeployment() *appsv1.Deployment {
	spec := j.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        j.name,
			Namespace:   j.namespace,
			Labels:      j.labels,
			Annotations: j.Annotations,
		},
		Spec: spec,
	}
}

func (j *Jigasi) prepareDeploymentSpec() appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: j.labels,
		},
		Replicas: &j.Spec.Replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: j.labels,
			},
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: &j.Spec.TerminationGracePeriodSeconds,
				ImagePullSecrets:              j.Spec.ImagePullSecrets,
				Containers: []corev1.Container{
					{
						Name:            name,
						Image:           j.Spec.Image,
						ImagePullPolicy: j.Spec.ImagePullPolicy,
						Env:             j.Spec.Environments,
						Ports:           jitsi.GetContainerPorts(j.Spec.Ports),
						Resources:       j.Spec.Resources,
						SecurityContext: &j.Spec.SecurityContext,
					},
				},
			},
		},
	}
}

func (j *Jigasi) Update(deployment *appsv1.Deployment) error {
	deployment.Annotations = j.Annotations
	deployment.Labels = j.Labels
	deployment.Spec = j.prepareDeploymentSpec()
	return j.Client.Update(j.ctx, deployment)
}

func (j *Jigasi) Delete() error {
	if err := utils.RemoveFinalizer(j.ctx, j.Client, j.Jigasi); err != nil {
		j.log.Info("can't remove finalizer", "error", err)
	}
	deployment, err := j.Get()
	if err != nil {
		return err
	}
	err = j.Client.Delete(j.ctx, deployment)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (j *Jigasi) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := j.Client.Get(j.ctx, types.NamespacedName{
		Namespace: j.namespace,
		Name:      j.name,
	}, deployment)
	return deployment, err
}
