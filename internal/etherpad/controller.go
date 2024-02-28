// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package etherpad

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/etherpad/v1alpha2"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconcile struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Etherpad{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=meeting.ko,resources=etherpads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=meeting.ko,resources=etherpads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=meeting.ko,resources=etherpads/finalizers,verbs=update

func (r *Reconcile) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("etherpad", req.NamespacedName)

	etherpad, err := newInstance(ctx, r.Client, reqLogger, req)
	if err != nil {
		if meetingerr.IsUnderDeletion(err) {
			if delErr := etherpad.Delete(); client.IgnoreNotFound(delErr) != nil {
				return ctrl.Result{}, delErr
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if updErr := etherpad.Update(); updErr != nil {
		if apierrors.IsNotFound(updErr) {
			if createErr := etherpad.Create(); createErr != nil {
				return ctrl.Result{}, createErr
			}
		} else {
			reqLogger.Info("failed to update etherpad deployment", "error", updErr)
			return ctrl.Result{}, updErr
		}
	}
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}
