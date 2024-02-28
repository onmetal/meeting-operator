// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
