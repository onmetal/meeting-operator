// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package jvb

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
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

type JVB struct {
	client.Client
	*v1beta1.JVB

	ctx           context.Context
	log           logr.Logger
	envs          []corev1.EnvVar
	replicaName   string
	replica, port int32
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.JVB{}).
		WithEventFilter(r.constructPredicates()).
		Complete(r)
}

func (r *Reconciler) constructPredicates() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}
}

// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jvbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jvbs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jvbs/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)

	jvb, err := r.newInstance(ctx, reqLogger, req)
	if err != nil {
		if meeterr.IsUnderDeletion(err) {
			reqLogger.Info("reconciliation finished")
			return ctrl.Result{}, jvb.Delete()
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if createErr := jvb.Create(); createErr != nil {
		return ctrl.Result{}, createErr
	}
	if updErr := jvb.Update(); updErr != nil {
		return ctrl.Result{}, updErr
	}
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) newInstance(ctx context.Context, l logr.Logger, req ctrl.Request) (*JVB, error) {
	j := &v1beta1.JVB{}
	if err := r.Get(ctx, req.NamespacedName, j); err != nil {
		return nil, err
	}
	var jvbExternalPort int32 = 10000
	if j.Spec.Port.Port != 0 {
		jvbExternalPort = j.Spec.Port.Port
	}
	if !j.DeletionTimestamp.IsZero() {
		return &JVB{
			Client: r.Client,
			JVB:    j,
			ctx:    ctx,
			log:    l,
			port:   jvbExternalPort,
		}, meeterr.UnderDeletion()
	}
	if err := utils.AddFinalizer(ctx, r.Client, j); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &JVB{
		Client: r.Client,
		JVB:    j,
		envs:   j.Spec.Environments,
		ctx:    ctx,
		log:    l,
		port:   jvbExternalPort,
	}, nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oldObj, oldOk := e.ObjectOld.(*v1beta1.JVB)
	newObj, newOk := e.ObjectNew.(*v1beta1.JVB)
	if !oldOk || !newOk {
		return false
	}
	if len(oldObj.Finalizers) < 1 && len(newObj.Finalizers) >= 1 {
		return false
	}
	if !reflect.DeepEqual(oldObj.Status, newObj.Status) {
		return false
	}
	return true
}
