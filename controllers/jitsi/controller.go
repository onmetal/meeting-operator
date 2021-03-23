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

package jitsi

import (
	"context"
	"github.com/go-logr/logr"
	jitsiv1alpha1 "github.com/onmetal/meeting-operator/api/v1alpha1"
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	jitsiFinalizer = "jitsi.finalizers.meeting.ko"
)

//+kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jitsi,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=configmaps,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jitsi/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jitsi/finalizers,verbs=update
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("jitsi", req.NamespacedName)

	jitsi := &jitsiv1alpha1.Jitsi{}
	if err := r.Get(ctx, req.NamespacedName, jitsi); err != nil {
		r.Log.Info("unable to fetch Jitsi", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if jitsi.ObjectMeta.DeletionTimestamp.IsZero() {
		if !utils.ContainsString(jitsi.ObjectMeta.Finalizers, jitsiFinalizer) {
			jitsi.ObjectMeta.Finalizers = append(jitsi.ObjectMeta.Finalizers, jitsiFinalizer)
			if err := r.Update(context.Background(), jitsi); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if utils.ContainsString(jitsi.ObjectMeta.Finalizers, jitsiFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(jitsi); err != nil {
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			jitsi.ObjectMeta.Finalizers = utils.RemoveString(jitsi.ObjectMeta.Finalizers, jitsiFinalizer)
			if err := r.Update(context.Background(), jitsi); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info("external resources were deleted")
		}
		return ctrl.Result{}, nil
	}
	if err := r.makeWeb(ctx, jitsi); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.makeJicofo(ctx, jitsi); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.makeProsody(ctx, jitsi); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.makeJVB(ctx, jitsi); err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jitsiv1alpha1.Jitsi{}).
		Owns(&appsv1.Deployment{}).
		Owns(&v1.Service{}).
		Complete(r)
}

func (r *Reconciler) deleteExternalResources(jitsi *jitsiv1alpha1.Jitsi) error {
	ctx := context.Background()
	if err := r.cleanupWebObjects(ctx, jitsi); err != nil {
		return err
	}
	if err := r.cleanupProsodyObjects(ctx, jitsi); err != nil {
		return err
	}
	if err := r.cleanupJicofoObjects(ctx, jitsi); err != nil {
		return err
	}
	if err := r.cleanUpJVBObjects(ctx, jitsi); err != nil {
		return err
	}
	return nil
}
