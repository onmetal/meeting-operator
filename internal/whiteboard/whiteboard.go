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
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/whiteboard/v1alpha1"
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WhiteBoard interface {
	Create() error
	Update() error
	Delete() error
}

type Board struct {
	client.Client
	*v1alpha1.WhiteBoardSpec

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

func NewWhiteboard(ctx context.Context, w *v1alpha1.WhiteBoard,
	c client.Client, l logr.Logger) (WhiteBoard, error) {
	labels := utils.GetDefaultLabels(w.Name)
	return &Board{
		Client:         c,
		WhiteBoardSpec: &w.Spec,
		ctx:            ctx,
		log:            l,
		name:           w.Name,
		namespace:      w.Namespace,
		labels:         labels,
	}, nil
}

func (w *Board) Create() error {
	preparedDeployment := w.prepareDeployment()
	return w.Client.Create(w.ctx, preparedDeployment)
}

func (w *Board) prepareDeployment() *appsv1.Deployment {
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

func (w *Board) prepareDeploymentSpec() appsv1.DeploymentSpec {
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
				ImagePullSecrets: w.WhiteBoardSpec.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            w.name,
						Image:           w.Image,
						ImagePullPolicy: w.ImagePullPolicy,
						Env:             w.Environments,
						Ports:           w.getContainerPorts(),
					},
				},
			},
		},
	}
}

func (w *Board) getContainerPorts() []v1.ContainerPort {
	var ports []v1.ContainerPort
	for svc := range w.Services {
		ports = append(ports, v1.ContainerPort{
			Name:          w.Services[svc].PortName,
			ContainerPort: w.Services[svc].Port,
			Protocol:      w.Services[svc].Protocol,
		})
	}
	return ports
}

func (w *Board) Update() error {
	updatedDeployment := w.prepareDeployment()
	return w.Client.Update(w.ctx, updatedDeployment)
}

func (w *Board) Delete() error {
	deployment, err := w.Get()
	if err != nil {
		return err
	}
	return w.Client.Delete(w.ctx, deployment)
}

func (w *Board) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := w.Client.Get(w.ctx, types.NamespacedName{
		Namespace: w.namespace,
		Name:      w.name,
	}, deployment)
	return deployment, err
}
