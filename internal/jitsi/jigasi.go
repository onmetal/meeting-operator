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
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const JigasiName = "jigasi"

type Jigasi struct {
	client.Client
	*v1beta1.Jigasi

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

func NewJigasi(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (Jitsi, error) {
	j := &v1beta1.Jigasi{}
	if err := c.Get(ctx, req.NamespacedName, j); err != nil {
		return nil, err
	}
	defaultLabels := utils.GetDefaultLabels(JigasiName)
	if !j.DeletionTimestamp.IsZero() {
		return &Jigasi{
			Client:    c,
			Jigasi:    j,
			name:      JigasiName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    defaultLabels,
		}, meeterr.UnderDeletion()
	}
	if err := addFinalizerToJigasi(ctx, c, j); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &Jigasi{
		Client:    c,
		Jigasi:    j,
		ctx:       ctx,
		log:       l,
		name:      JigasiName,
		namespace: j.Namespace,
		labels:    defaultLabels,
	}, nil
}

func addFinalizerToJigasi(ctx context.Context, c client.Client, j *v1beta1.Jigasi) error {
	if utils.ContainsString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		return nil
	}
	j.ObjectMeta.Finalizers = append(j.ObjectMeta.Finalizers, utils.MeetingFinalizer)
	return c.Update(ctx, j)
}

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
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: j.labels,
			},
			Spec: v1.PodSpec{
				TerminationGracePeriodSeconds: &j.Spec.TerminationGracePeriodSeconds,
				ImagePullSecrets:              j.Spec.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            JigasiName,
						Image:           j.Spec.Image,
						ImagePullPolicy: j.Spec.ImagePullPolicy,
						Env:             j.Spec.Environments,
						Ports:           getContainerPorts(j.Spec.Ports),
						Resources:       j.Spec.Resources,
						SecurityContext: &j.Spec.SecurityContext,
					},
				},
			},
		},
	}
}

func (j *Jigasi) Update() error {
	updatedDeployment := j.prepareDeployment()
	return j.Client.Update(j.ctx, updatedDeployment)
}

func (j *Jigasi) UpdateStatus() error { return nil }

func (j *Jigasi) Delete() error {
	if err := j.removeFinalizerFromJigasi(); err != nil {
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

func (j *Jigasi) removeFinalizerFromJigasi() error {
	if !utils.ContainsString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		return nil
	}
	j.ObjectMeta.Finalizers = utils.RemoveString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer)
	return j.Client.Update(j.ctx, j.Jigasi)
}

func (j *Jigasi) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := j.Client.Get(j.ctx, types.NamespacedName{
		Namespace: j.namespace,
		Name:      j.name,
	}, deployment)
	return deployment, err
}
