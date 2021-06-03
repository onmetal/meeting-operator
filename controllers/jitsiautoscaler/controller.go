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

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsiautoscaler/v1alpha1"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/jitsiautoscaler"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueAfterSecond = 600 * time.Second

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

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("jitsi autoscaler", req.NamespacedName)
	jas, err := jitsiautoscaler.New(ctx, r.Client, reqLogger, req)
	if err != nil {
		if meetingerr.IsUnderDeletion(err) {
			return ctrl.Result{}, jas.Stop()
		}
		return ctrl.Result{}, err
	}
	if watchErr := jas.Watch(); watchErr != nil {
		return ctrl.Result{}, watchErr
	}
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{RequeueAfter: requeueAfterSecond}, nil
}
