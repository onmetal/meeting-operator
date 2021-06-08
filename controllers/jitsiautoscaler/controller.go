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

	meetingerr "github.com/onmetal/meeting-operator/internal/errors"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsiautoscaler/v1alpha1"
	"github.com/onmetal/meeting-operator/internal/jitsiautoscaler"
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
	jas, err := jitsiautoscaler.New(ctx, r.Client, reqLogger, req)
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
