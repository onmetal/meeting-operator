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
	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/jitsi"
	"github.com/onmetal/meeting-operator/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type JVBReconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *JVBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.JVB{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=meeting.ko,resources=jitsis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configmaps,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meeting.ko,resources=jitsis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=meeting.ko,resources=jitsis/finalizers,verbs=update

func (r *JVBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("jitsi", req.NamespacedName)

	j := &v1beta1.JVB{}
	if err := r.Get(ctx, req.NamespacedName, j); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !j.DeletionTimestamp.IsZero() {
		r.onDelete(ctx, j)
		return ctrl.Result{}, nil
	}
	r.makeJVB(ctx, j)
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *JVBReconciler) makeJVB(ctx context.Context, j *v1beta1.JVB) {
	if j.Status.Replicas > j.Spec.Replicas {
		replica := j.Status.Replicas
		delta := j.Status.Replicas - j.Spec.Replicas
		for replica >= delta && replica != 1 {
			jts := jitsi.NewJVB(ctx, replica, j, r.Client, r.Log)
			if delErr := jts.Delete(); delErr != nil {
				r.Log.Info("can't scale down jvb replicas", "error", delErr)
			}
			replica--
		}
	}
	for replica := int32(1); replica <= j.Spec.Replicas; replica++ {
		jts := jitsi.NewJVB(ctx, replica, j, r.Client, r.Log)
		if createErr := jts.Create(); createErr != nil {
			if meetingerr.IsAlreadyExists(createErr) {
				if updErr := jts.Update(); updErr != nil {
					r.Log.Info("failed to update jvb instance", "error", updErr)
				}
				continue
			}
			r.Log.Info("failed to create jvb", "error", createErr)
			continue
		}
	}
	j.Status.Replicas = j.Spec.Replicas
	if err := r.Client.Status().Update(ctx, j); err != nil {
		r.Log.Info("can't update jitsi custom resource status", "error", err)
	}
}

func (r *JVBReconciler) onDelete(ctx context.Context, j *v1beta1.JVB) {
	r.deleteJVB(ctx, j)
	if utils.ContainsString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		j.ObjectMeta.Finalizers = utils.RemoveString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer)
		if err := r.Client.Update(ctx, j); err != nil {
			r.Log.Info("can't update jitsi cr", "error", err)
		}
	}
}

func (r *JVBReconciler) deleteJVB(ctx context.Context, j *v1beta1.JVB) {
	for replica := int32(1); replica <= j.Spec.Replicas; replica++ {
		jvb := jitsi.NewJVB(ctx, replica, j, r.Client, r.Log)
		if err := jvb.Delete(); client.IgnoreNotFound(err) != nil {
			r.Log.Info("failed to delete jvb instance", "error", err)
		}
	}
}
