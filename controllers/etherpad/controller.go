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

package etherpad

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	etherpadv1alpha1 "github.com/onmetal/meeting-operator/api/v1alpha1"
)

type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	etherpadFinalizer = "etherpad.finalizers.meeting.ko"
)

//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads/finalizers,verbs=update
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("etherpad", req.NamespacedName)

	etherpad := &etherpadv1alpha1.Etherpad{}
	if err := r.Get(ctx, req.NamespacedName, etherpad); err != nil {
		r.Log.Info("unable to fetch Etherpad", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if etherpad.ObjectMeta.DeletionTimestamp.IsZero() {
		if !utils.ContainsString(etherpad.ObjectMeta.Finalizers, etherpadFinalizer) {
			etherpad.ObjectMeta.Finalizers = append(etherpad.ObjectMeta.Finalizers, etherpadFinalizer)
			if err := r.Update(context.Background(), etherpad); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if utils.ContainsString(etherpad.ObjectMeta.Finalizers, etherpadFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(etherpad); err != nil {
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			etherpad.ObjectMeta.Finalizers = utils.RemoveString(etherpad.ObjectMeta.Finalizers, etherpadFinalizer)
			if err := r.Update(context.Background(), etherpad); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info("external resources were deleted")
		}
		return ctrl.Result{}, nil
	}
	if err := r.make(ctx, etherpad); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etherpadv1alpha1.Etherpad{}).
		Owns(&appsv1.Deployment{}).
		Owns(&v1.Service{}).
		Complete(r)
}

func (r *Reconciler) deleteExternalResources(etherpad *etherpadv1alpha1.Etherpad) error {
	ctx := context.Background()
	if err := r.cleanUpEtherpadObjects(ctx, etherpad); err != nil {
		return err
	}
	return nil
}


func (r *Reconciler) createFinalizer(etherpad *etherpadv1alpha1.Etherpad) error {
	if !utils.ContainsString(etherpad.ObjectMeta.Finalizers, etherpadFinalizer) {
		etherpad.ObjectMeta.Finalizers = append(etherpad.ObjectMeta.Finalizers, etherpadFinalizer)
		if err := r.Update(context.Background(), etherpad); err != nil {
			return err
		}
	}
	return nil
}
