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
	"github.com/onmetal/meeting-operator/internal/jitsi"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	jitsiFinalizer = "jitsi.finalizers.meeting.ko"
)

type Reconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Jitsi{}).
		WithEventFilter(r.predicateFuncs()).
		Complete(r)
}

func (r *Reconciler) predicateFuncs() predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: r.onDelete,
	}
}

//+kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jitsi,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=configmaps,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jitsi/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jitsi/finalizers,verbs=update
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("jitsi", req.NamespacedName)

	j := &v1alpha1.Jitsi{}
	if err := r.Get(ctx, req.NamespacedName, j); err != nil {
		r.Log.Info("unable to fetch Jitsi", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//// examine DeletionTimestamp to determine if object is under deletion
	//if jitsi.ObjectMeta.DeletionTimestamp.IsZero() {
	//	if !utils.ContainsString(jitsi.ObjectMeta.Finalizers, jitsiFinalizer) {
	//		jitsi.ObjectMeta.Finalizers = append(jitsi.ObjectMeta.Finalizers, jitsiFinalizer)
	//		if err := r.Update(context.Background(), jitsi); err != nil {
	//			return ctrl.Result{}, err
	//		}
	//	}
	//} else {
	//	// The object is being deleted
	//	if utils.ContainsString(jitsi.ObjectMeta.Finalizers, jitsiFinalizer) {
	//		// our finalizer is present, so lets handle any external dependency
	//		if err := r.deleteExternalResources(jitsi); err != nil {
	//			return ctrl.Result{}, err
	//		}
	//		// remove our finalizer from the list and update it.
	//		jitsi.ObjectMeta.Finalizers = utils.RemoveString(jitsi.ObjectMeta.Finalizers, jitsiFinalizer)
	//		if err := r.Update(context.Background(), jitsi); err != nil {
	//			return ctrl.Result{}, err
	//		}
	//		r.Log.Info("external resources were deleted")
	//	}
	//	return ctrl.Result{}, nil
	//}
	if err := r.make(ctx, "web", j); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.make(ctx, "jicofo", j); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.make(ctx, "prosody", j); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.makeJVB(ctx, j); err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) make(ctx context.Context, appName string, j *v1alpha1.Jitsi) error {
	jts := jitsi.NewJitsi(ctx, appName, j, r.Client, r.Log)
	if err := jts.Update(); err != nil {
		if apierrors.IsNotFound(err) {
			if err := jts.Create(); err != nil {
				return err
			}
		} else {
			r.Log.Info("failed to update jitsi deployment", "error", err)
			return err
		}
	}
	svc := jitsi.NewService(ctx, appName, j, r.Client, r.Log)
	if err := svc.Update(); err != nil {
		if apierrors.IsNotFound(err) {
			if err := svc.Create(); err != nil {
				return err
			}
		} else {
			r.Log.Info("failed to update jitsi service", "error", err)
			return err
		}
	}
	return nil
}

func (r *Reconciler) makeJVB(ctx context.Context, j *v1alpha1.Jitsi) error {
	jts := jitsi.NewJitsi(ctx, "jvb", j, r.Client, r.Log)
	if err := jts.Update(); err != nil {
		r.Log.Info("failed to update jitsi deployment", "error", err)
		return err
	}
	return nil
}

func (r *Reconciler) onDelete(e event.DeleteEvent) bool {
	jitsiObj, ok := e.Object.(*v1alpha1.Jitsi)
	if !ok {
		return false
	}
	ctx := context.Background()
	if err := r.deleteComponents(ctx, "web", jitsiObj); err != nil {
		r.Log.Info("failed to delete web component", "error", err)
	}
	if err := r.deleteComponents(ctx, "jicofo", jitsiObj); err != nil {
		r.Log.Info("failed to delete jicofo component", "error", err)

	}
	if err := r.deleteComponents(ctx, "prosody", jitsiObj); err != nil {
		r.Log.Info("failed to delete prosody component", "error", err)
	}
	if err := r.deleteJVB(ctx, jitsiObj); err != nil {
		r.Log.Info("failed to delete jvb component", "error", err)
	}
	return false
}

func (r *Reconciler) deleteComponents(ctx context.Context, appName string, j *v1alpha1.Jitsi) error {
	jts := jitsi.NewJitsi(ctx, appName, j, r.Client, r.Log)
	if err := jts.Delete(); err != nil {
		return err
	}
	svc := jitsi.NewService(ctx, appName, j, r.Client, r.Log)
	if err := svc.Delete(); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) deleteJVB(ctx context.Context, j *v1alpha1.Jitsi) error {
	jts := jitsi.NewJitsi(ctx, "jvb", j, r.Client, r.Log)
	if err := jts.Delete(); err != nil {
		return err
	}
	return nil
}
