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

	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/whiteboard/v1alpha2"
	"github.com/onmetal/meeting-operator/internal/whiteboard"
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
		For(&v1alpha2.WhiteBoard{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=meeting.ko,resources=whiteboards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=meeting.ko,resources=whiteboards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=meeting.ko,resources=whiteboards/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("whiteboard", req.NamespacedName)

	wb, err := whiteboard.New(ctx, r.Client, reqLogger, req)
	if err != nil {
		if meetingerr.IsUnderDeletion(err) {
			if delErr := wb.Delete(); delErr != nil {
				return ctrl.Result{}, delErr
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if updErr := wb.Update(); updErr != nil {
		if apierrors.IsNotFound(updErr) {
			if createErr := wb.Create(); createErr != nil {
				return ctrl.Result{}, createErr
			}
		}
	}
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}
