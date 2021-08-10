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

func NewWeb(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (Jitsi, error) {
	w := &v1beta1.Web{}
	if err := c.Get(ctx, req.NamespacedName, w); err != nil {
		return nil, err
	}
	labels := utils.GetDefaultLabels(WebName)
	s := newService(ctx, c, l, WebName, w.Namespace, w.Spec.ServiceAnnotations, labels, w.Spec.ServiceType, w.Spec.Ports)
	if !w.DeletionTimestamp.IsZero() {
		return &Web{
			Client:    c,
			Web:       w,
			service:   s,
			ctx:       ctx,
			log:       l,
			name:      WebName,
			namespace: w.Namespace,
			labels:    labels,
		}, meeterr.UnderDeletion()
	}
	if err := addFinalizerToWeb(ctx, c, w); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &Web{
		Client:    c,
		Web:       w,
		service:   s,
		ctx:       ctx,
		log:       l,
		name:      WebName,
		namespace: w.Namespace,
		labels:    labels,
	}, nil
	//case ProsodyName:
	//	labels := utils.GetDefaultLabels(ProsodyName)
	//	return &Prosody{
	//		Client:    c,
	//		Prosody:   &w.Spec.Prosody,
	//		name:      ProsodyName,
	//		namespace: w.Namespace,
	//		ctx:       ctx,
	//		log:       l,
	//		labels:    labels,
	//	}, nil
	//case JicofoName:
	//	labels := utils.GetDefaultLabels(JicofoName)
	//	return &Jicofo{
	//		Client:    c,
	//		Jicofo:    &w.Spec.Jicofo,
	//		name:      JicofoName,
	//		namespace: w.Namespace,
	//		ctx:       ctx,
	//		log:       l,
	//		labels:    labels,
	//	}, nil
	//case JibriName:
	//	labels := utils.GetDefaultLabels(JibriName)
	//	return &Jibri{
	//		Client:    c,
	//		Jibri:     &w.Spec.Jibri,
	//		name:      JibriName,
	//		namespace: w.Namespace,
	//		ctx:       ctx,
	//		log:       l,
	//		labels:    labels,
	//	}, nil
	//case JigasiName:
	//	labels := utils.GetDefaultLabels(JigasiName)
	//	return &Jigasi{
	//		Client:    c,
	//		Jigasi:    &w.Spec.Jigasi,
	//		name:      JigasiName,
	//		namespace: w.Namespace,
	//		ctx:       ctx,
	//		log:       l,
	//		labels:    labels,
	//	}, nil
}

func addFinalizerToWeb(ctx context.Context, c client.Client, j *v1beta1.Web) error {
	if utils.ContainsString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		return nil
	}
	j.ObjectMeta.Finalizers = append(j.ObjectMeta.Finalizers, utils.MeetingFinalizer)
	return c.Update(ctx, j)
}

func (w *Web) Create() error {
	if err := w.service.Create(); err != nil {
		return err
	}
	preparedDeployment := w.prepareDeployment()
	return w.Client.Create(w.ctx, preparedDeployment)
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
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: w.labels,
			},
			Spec: v1.PodSpec{
				TerminationGracePeriodSeconds: &w.Spec.TerminationGracePeriodSeconds,
				ImagePullSecrets:              w.Spec.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            WebName,
						Image:           w.Spec.Image,
						ImagePullPolicy: w.Spec.ImagePullPolicy,
						Env:             w.Spec.Environments,
						Ports:           getContainerPorts(w.Spec.Ports),
						Resources:       w.Spec.Resources,
						SecurityContext: &w.Spec.SecurityContext,
					},
				},
			},
		},
	}
}

func (w *Web) Update() error {
	if err := w.service.Update(); err != nil {
		return err
	}
	updatedDeployment := w.prepareDeployment()
	return w.Client.Update(w.ctx, updatedDeployment)
}

func (w *Web) Delete() error {
	if err := w.service.Delete(); err != nil {
		return err
	}
	deployment, err := w.Get()
	if err != nil {
		return err
	}
	if utils.ContainsString(w.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		w.ObjectMeta.Finalizers = utils.RemoveString(w.ObjectMeta.Finalizers, utils.MeetingFinalizer)
		if err := w.Client.Update(w.ctx, w); err != nil {
			w.log.Info("can't update web cr", "error", err)
		}
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
