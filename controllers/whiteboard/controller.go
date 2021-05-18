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

	"github.com/onmetal/meeting-operator/apis/whiteboard/v1alpha1"
	"github.com/onmetal/meeting-operator/internal/whiteboard"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
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
		For(&v1alpha1.WhiteBoard{}).
		WithEventFilter(r.constructPredicates()).
		Complete(r)
}

func (r *Reconciler) constructPredicates() predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: r.onDelete,
	}
}

//+kubebuilder:rbac:groups=meeting.ko,resources=whiteboards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=meeting.ko,resources=whiteboards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=meeting.ko,resources=whiteboards/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("whiteboard", req.NamespacedName)

	w := &v1alpha1.WhiteBoard{}
	if err := r.Get(ctx, req.NamespacedName, w); err != nil {
		return ctrl.Result{}, err
	}
	wb, err := whiteboard.NewWhiteboard(ctx, w, r.Client, r.Log)
	if err != nil {
		r.Log.Info("cannot create new instance of whiteboard", "error", err)
		return ctrl.Result{}, err
	}
	svc := whiteboard.NewService(ctx, w, r.Client, r.Log)
	if err := wb.Update(); err != nil {
		if apierrors.IsNotFound(err) {
			if err := wb.Create(); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	if err := svc.Update(); err != nil {
		if apierrors.IsNotFound(err) {
			if err := svc.Create(); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	r.Log.Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) onDelete(e event.DeleteEvent) bool {
	deletedObj, ok := e.Object.(*v1alpha1.WhiteBoard)
	if !ok {
		return false
	}
	ctx := context.Background()
	wb, err := whiteboard.NewWhiteboard(ctx, deletedObj, r.Client, r.Log)
	if err != nil {
		r.Log.Info("cannot create new instance of whiteboard", "error", err)
		return false
	}
	if err := wb.Delete(); err != nil {
		r.Log.Error(err, "failed to delete whiteboard deployment")
	}
	svc := whiteboard.NewService(ctx, deletedObj, r.Client, r.Log)
	if err := svc.Delete(); err != nil {
		r.Log.Error(err, "failed to delete whiteboard deployment")
	}
	return false
}
