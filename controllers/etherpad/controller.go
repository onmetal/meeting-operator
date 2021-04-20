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

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/etherpad/v1alpha1"
	"github.com/onmetal/meeting-operator/internal/etherpad"
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
		For(&v1alpha1.Etherpad{}).
		WithEventFilter(r.constructPredicates()).
		Complete(r)
}

func (r *Reconciler) constructPredicates() predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: r.onDelete,
	}
}

//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads/finalizers,verbs=update
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("etherpad", req.NamespacedName)

	eth := &v1alpha1.Etherpad{}
	if err := r.Get(ctx, req.NamespacedName, eth); err != nil {
		r.Log.Error(err, "failed to get etherpad")
		return ctrl.Result{}, err
	}

	newEtherpad := etherpad.NewDeployment(ctx, r.Client, r.Log, eth)
	if err := newEtherpad.Update(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := newEtherpad.Create(); createErr != nil {
				return ctrl.Result{}, createErr
			}
		} else {
			r.Log.Info("failed to update etherpad deployment", "error", err)
			return ctrl.Result{}, err
		}
	}

	newEtherpadSvc := etherpad.NewService(ctx, r.Client, r.Log, eth)
	if err := newEtherpadSvc.Update(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := newEtherpadSvc.Create(); createErr != nil {
				return ctrl.Result{}, createErr
			}
		} else {
			r.Log.Info("failed to update etherpad service", "error", err)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) onDelete(e event.DeleteEvent) bool {
	deletedEtherpadObj, ok := e.Object.(*v1alpha1.Etherpad)
	if !ok {
		return false
	}
	ctx := context.Background()
	eth := etherpad.NewDeployment(ctx, r.Client, r.Log, deletedEtherpadObj)
	if err := eth.Delete(); err != nil {
		r.Log.Error(err, "failed to delete etherpad deployment")
	}
	svc := etherpad.NewService(ctx, r.Client, r.Log, deletedEtherpadObj)
	if err := svc.Delete(); err != nil {
		r.Log.Error(err, "failed to delete etherpad deployment")
	}
	return false
}
