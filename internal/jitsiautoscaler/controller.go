// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package jitsiautoscaler

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsiautoscaler/v1alpha1"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AutoScaler{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=meeting.ko,resources=autoscalers,verbs=get;list;watch;update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("jitsi autoscaler", req.NamespacedName)
	jas, err := newInstance(ctx, r.Client, reqLogger, req)
	if err != nil {
		if meetingerr.IsNotExist(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	jas.Scale()
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{RequeueAfter: jas.Repeat()}, nil
}
