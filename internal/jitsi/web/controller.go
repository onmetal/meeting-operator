// /*
// Copyright (c) 2021 T-Systems International GmbH, SAP SE or an SAP affiliate company. All right reserved
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package web

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/jitsi"
	"github.com/onmetal/meeting-operator/internal/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type Reconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

type Web struct {
	client.Client
	*v1beta1.Web

	Service         jitsi.Servicer
	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Web{}).
		WithEventFilter(r.constructPredicates()).
		Complete(r)
}

func (r *Reconciler) constructPredicates() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: isSpecUpdated,
		DeleteFunc: func(event.DeleteEvent) bool { return false },
	}
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=webs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=webs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=webs/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)

	web, err := r.newInstance(ctx, reqLogger, req)
	if err != nil {
		if meeterr.IsUnderDeletion(err) {
			reqLogger.V(1).Info("deletion finished")
			return ctrl.Result{}, web.Delete()
		}
		reqLogger.Error(err, "can't create new instance of web")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	deployment, getErr := web.Get()
	if apierrors.IsNotFound(getErr) {
		if createErr := web.Create(); createErr != nil {
			return ctrl.Result{}, createErr
		}
		reqLogger.V(1).Info("reconciliation finished")
		return ctrl.Result{}, nil
	}

	if updErr := web.Update(deployment); updErr != nil {
		return ctrl.Result{}, updErr
	}

	reqLogger.V(1).Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) newInstance(ctx context.Context, l logr.Logger, req ctrl.Request) (*Web, error) {
	w := &v1beta1.Web{}
	if err := r.Get(ctx, req.NamespacedName, w); err != nil {
		return nil, err
	}
	defaultLabels := utils.GetDefaultLabelsForApp(name)
	s := jitsi.NewService(ctx, r.Client, l, name, w.Namespace, w.Spec.ServiceAnnotations, defaultLabels, w.Spec.ServiceType, w.Spec.Ports)
	if !w.DeletionTimestamp.IsZero() {
		return &Web{
			Client:    r.Client,
			Web:       w,
			Service:   s,
			ctx:       ctx,
			log:       l,
			name:      name,
			namespace: w.Namespace,
			labels:    defaultLabels,
		}, meeterr.UnderDeletion()
	}
	if err := utils.AddFinalizer(ctx, r.Client, w); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &Web{
		Client:    r.Client,
		Web:       w,
		Service:   s,
		ctx:       ctx,
		log:       l,
		name:      name,
		namespace: w.Namespace,
		labels:    defaultLabels,
	}, nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oldObj, oldOk := e.ObjectOld.(*v1beta1.Web)
	newObj, newOk := e.ObjectNew.(*v1beta1.Web)
	if !oldOk || !newOk {
		return false
	}
	if len(oldObj.Finalizers) < 1 && len(newObj.Finalizers) >= 1 {
		return false
	}
	return true
}
