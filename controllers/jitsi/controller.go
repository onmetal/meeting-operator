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

var jitsiServices = []string{jitsi.WebName, jitsi.ProsodyName, jitsi.JicofoName, jitsi.JibriName}

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

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=meeting.ko,resources=jitsis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configmaps,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meeting.ko,resources=jitssi/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=meeting.ko,resources=jitsis/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("jitsi", req.NamespacedName)

	j := &v1alpha1.Jitsi{}
	if err := r.Get(ctx, req.NamespacedName, j); err != nil {
		r.Log.Info("unable to fetch Jitsi", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	for _, appName := range jitsiServices {
		if err := r.make(ctx, appName, j); err != nil {
			return ctrl.Result{}, err
		}
	}
	r.makeJVB(ctx, j)
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) make(ctx context.Context, appName string, j *v1alpha1.Jitsi) error {
	if !shouldContinue(appName, j) {
		return nil
	}
	jts, err := jitsi.New(ctx, appName, j, r.Client, r.Log)
	if err != nil {
		return err
	}
	if err := jts.Update(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := jts.Create(); createErr != nil {
				return createErr
			}
		} else {
			r.Log.Info("failed to update jitsi deployment", "error", err)
			return err
		}
	}
	svc := jitsi.NewService(ctx, appName, j, r.Client, r.Log)
	if err := svc.Update(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := svc.Create(); createErr != nil {
				return createErr
			}
		} else {
			r.Log.Info("failed to update jitsi service", "error", err)
			return err
		}
	}
	return nil
}

func (r *Reconciler) makeJVB(ctx context.Context, j *v1alpha1.Jitsi) {
	if !shouldContinue(jitsi.JvbName, j) {
		return
	}
	for replica := int32(1); replica <= j.Spec.JVB.Replicas; replica++ {
		jts := jitsi.NewJVB(ctx, replica, j, r.Client, r.Log)
		if createErr := jts.Create(); createErr != nil {
			if apierrors.IsAlreadyExists(createErr) {
				if updErr := jts.Update(); updErr != nil {
					r.Log.Info("failed to update jvb instance", "error", updErr)
				}
				continue
			}
			r.Log.Info("failed to create jvb", "error", createErr)
			continue
		}
	}
}

func (r *Reconciler) onDelete(e event.DeleteEvent) bool {
	ctx := context.Background()
	jitsiObj, ok := e.Object.(*v1alpha1.Jitsi)
	if !ok {
		return false
	}
	for _, appName := range jitsiServices {
		if err := r.deleteComponents(ctx, appName, jitsiObj); err != nil {
			r.Log.Info("failed to delete component", "error", err)
		}
	}
	r.deleteJVB(ctx, jitsiObj)
	return false
}

func (r *Reconciler) deleteComponents(ctx context.Context, appName string, j *v1alpha1.Jitsi) error {
	jts, err := jitsi.New(ctx, appName, j, r.Client, r.Log)
	if err != nil {
		return err
	}
	if err := jts.Delete(); err != nil {
		return err
	}
	svc := jitsi.NewService(ctx, appName, j, r.Client, r.Log)
	if err := svc.Delete(); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) deleteJVB(ctx context.Context, j *v1alpha1.Jitsi) {
	for replica := int32(1); replica <= j.Spec.JVB.Replicas; replica++ {
		jts := jitsi.NewJVB(ctx, replica, j, r.Client, r.Log)
		if err := jts.Delete(); client.IgnoreNotFound(err) != nil {
			r.Log.Info("failed to delete jvb instance", "error", err)
		}
	}
}

func shouldContinue(appName string, j *v1alpha1.Jitsi) bool {
	switch appName {
	case jitsi.WebName:
		return j.Spec.Web.Replicas > 0
	case jitsi.ProsodyName:
		return j.Spec.Prosody.Replicas > 0
	case jitsi.JicofoName:
		return j.Spec.Jicofo.Replicas > 0
	case jitsi.JibriName:
		return j.Spec.Jibri.Replicas > 0
	case jitsi.JvbName:
		return j.Spec.JVB.Replicas > 0
	default:
		return false
	}
}
