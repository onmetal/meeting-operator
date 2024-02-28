// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package etherpad

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/etherpad/v1alpha2"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/utils"
	v1 "k8s.io/api/core/v1"
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

func newInstance(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (Etherpad, error) {
	eth := &v1alpha2.Etherpad{}
	if err := c.Get(ctx, req.NamespacedName, eth); err != nil {
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
	if err := utils.AddFinalizer(ctx, c, eth); err != nil {
		l.Info("can't add finalizer to etherpad", "error", err)
	}
	defaultLabels := utils.GetDefaultLabelsForApp(eth.Kind)
	return &etherpad{
		Client:   c,
		Etherpad: eth,
		ctx:      ctx,
		log:      l,
		labels:   defaultLabels,
	}, nil
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
