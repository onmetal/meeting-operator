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

package web

import (
	"github.com/onmetal/meeting-operator/internal/jitsi"
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const name = "web"

func (w *Web) Create() error {
	if svcErr := w.Service.Create(); svcErr != nil {
		return svcErr
	}
	newDeployment := w.prepareDeployment()
	return w.Client.Create(w.ctx, newDeployment)
}

func (w *Web) prepareDeployment() *appsv1.Deployment {
	spec := w.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        w.name,
			Namespace:   w.namespace,
			Labels:      w.labels,
			Annotations: w.Annotations,
		},
		Spec: spec,
	}
}

func (w *Web) prepareDeploymentSpec() appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: w.labels,
		},
		Replicas: &w.Spec.Replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: w.labels,
			},
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: &w.Spec.TerminationGracePeriodSeconds,
				ImagePullSecrets:              w.Spec.ImagePullSecrets,
				Containers: []corev1.Container{
					{
						Name:            name,
						Image:           w.Spec.Image,
						ImagePullPolicy: w.Spec.ImagePullPolicy,
						Env:             w.Spec.Environments,
						Ports:           jitsi.GetContainerPorts(w.Spec.Ports),
						Resources:       w.Spec.Resources,
						SecurityContext: &w.Spec.SecurityContext,
					},
				},
			},
		},
	}
}

func (w *Web) Update(deployment *appsv1.Deployment) error {
	if err := w.Service.Update(); err != nil {
		return err
	}
	deployment.Annotations = w.Annotations
	deployment.Labels = w.Labels
	deployment.Spec = w.prepareDeploymentSpec()
	return w.Client.Update(w.ctx, deployment)
}

func (w *Web) Delete() error {
	if err := utils.RemoveFinalizer(w.ctx, w.Client, w.Web); err != nil {
		w.log.Info("can't remove finalizer", "error", err)
	}
	if err := w.Service.Delete(); err != nil {
		return err
	}
	deployment, err := w.Get()
	if err != nil {
		return err
	}
	err = w.Client.Delete(w.ctx, deployment)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (w *Web) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := w.Client.Get(w.ctx, types.NamespacedName{
		Namespace: w.namespace,
		Name:      w.name,
	}, deployment)
	return deployment, err
}
