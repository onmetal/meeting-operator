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
	"context"

	v1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/etherpad/v1alpha2"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Etherpad interface {
	Create() error
	Update() error
	Delete() error
}

type etherpad struct {
	client.Client
	*v1alpha2.Etherpad

	ctx    context.Context
	log    logr.Logger
	labels map[string]string
}

type service struct {
	client.Client

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
	annotations     map[string]string
	serviceType     v1.ServiceType
	ports           []v1alpha2.Port
}

func New(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (Etherpad, error) {
	eth := &v1alpha2.Etherpad{}
	if err := c.Get(ctx, req.NamespacedName, eth); err != nil {
		l.Error(err, "can't get etherpad")
		return nil, err
	}
	if !eth.DeletionTimestamp.IsZero() {
		return &etherpad{
			Client:   c,
			Etherpad: eth,
			ctx:      ctx,
			log:      l,
		}, meetingerr.UnderDeletion()
	}
	if err := addFinalizer(ctx, c, eth); err != nil {
		l.Info("can't add finalizer to etherpad", "error", err)
	}
	defaultLabels := utils.GetDefaultLabels(eth.Kind)
	return &etherpad{
		Client:   c,
		Etherpad: eth,
		ctx:      ctx,
		log:      l,
		labels:   defaultLabels,
	}, nil
}

func addFinalizer(ctx context.Context, c client.Client, etherpad *v1alpha2.Etherpad) error {
	if !utils.ContainsString(etherpad.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		etherpad.ObjectMeta.Finalizers = append(etherpad.ObjectMeta.Finalizers, utils.MeetingFinalizer)
		return c.Update(ctx, etherpad)
	}
	return nil
}

func newService(e *etherpad) Etherpad {
	return &service{
		Client:      e.Client,
		ctx:         e.ctx,
		log:         e.log,
		name:        e.Name,
		namespace:   e.Namespace,
		labels:      e.labels,
		annotations: e.Spec.ServiceAnnotations,
		serviceType: e.Spec.ServiceType,
		ports:       e.Spec.Ports,
	}
}
