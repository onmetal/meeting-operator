// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package jibri

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
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

type Jibri struct {
	client.Client
	*v1beta1.Jibri

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Jibri{}).
		WithEventFilter(r.constructPredicates()).
		Complete(r)
}

func (r *Reconciler) constructPredicates() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: isSpecUpdated,
		DeleteFunc: func(event.DeleteEvent) bool { return false },
	}
}

// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jibris,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jibris/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jibris/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)

	jibri, err := r.newInstance(ctx, reqLogger, req)
	if err != nil {
		if meeterr.IsUnderDeletion(err) {
			reqLogger.V(1).Info("deletion finished")
			return ctrl.Result{}, jibri.Delete()
		}
		reqLogger.Error(err, "can't create new instance of jibri")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sts, getErr := jibri.Get()
	if apierrors.IsNotFound(getErr) {
		if createErr := jibri.Create(); createErr != nil {
			return ctrl.Result{}, createErr
		}
		reqLogger.V(1).Info("reconciliation finished")
		return ctrl.Result{}, nil
	}

	if updErr := jibri.Update(sts); updErr != nil {
		return ctrl.Result{}, updErr
	}

	reqLogger.V(1).Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) newInstance(ctx context.Context, l logr.Logger, req ctrl.Request) (*Jibri, error) {
	j := &v1beta1.Jibri{}
	if err := r.Get(ctx, req.NamespacedName, j); err != nil {
		return nil, err
	}
	defaultLabels := utils.GetDefaultLabelsForApp(appName)
	if !j.DeletionTimestamp.IsZero() {
		return &Jibri{
			Client:    r.Client,
			Jibri:     j,
			name:      appName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    defaultLabels,
		}, meeterr.UnderDeletion()
	}
	if err := utils.AddFinalizer(ctx, r.Client, j); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &Jibri{
		Client:    r.Client,
		Jibri:     j,
		ctx:       ctx,
		log:       l,
		name:      appName,
		namespace: j.Namespace,
		labels:    defaultLabels,
	}, nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oldObj, oldOk := e.ObjectOld.(*v1beta1.Jibri)
	newObj, newOk := e.ObjectNew.(*v1beta1.Jibri)
	if !oldOk || !newOk {
		return false
	}
	if len(oldObj.Finalizers) < 1 && len(newObj.Finalizers) >= 1 {
		return false
	}
	return true
}
