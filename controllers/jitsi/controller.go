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

	"github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/jitsi"
	"github.com/onmetal/meeting-operator/internal/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var jitsiServices = []string{jitsi.WebName, jitsi.ProsodyName, jitsi.JicofoName, jitsi.JibriName, jitsi.JigasiName}

type Reconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Jitsi{}).
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

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("jitsi", req.NamespacedName)

	j := &v1alpha1.Jitsi{}
	if err := r.Get(ctx, req.NamespacedName, j); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !j.DeletionTimestamp.IsZero() {
		r.onDelete(ctx, j)
		return ctrl.Result{}, nil
	}
	for _, appName := range jitsiServices {
		if err := r.make(ctx, appName, j); err != nil {
			reqLogger.Error(err, "can't install jitsi component", "name", appName)
			return ctrl.Result{}, err
		}
	}
	r.makeJVB(ctx, j)
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) make(ctx context.Context, appName string, j *v1alpha1.Jitsi) error {
	jts, err := jitsi.New(ctx, appName, j, r.Client, r.Log)
	if err != nil {
		return err
	}
	if jtsUpdErr := jts.Update(); jtsUpdErr != nil {
		if apierrors.IsNotFound(jtsUpdErr) {
			if createErr := jts.Create(); createErr != nil {
				return createErr
			}
		} else {
			r.Log.Info("failed to update jitsi deployment", "error", jtsUpdErr)
			return jtsUpdErr
		}
	}
	svc := jitsi.NewService(ctx, appName, j, r.Client, r.Log)
	if svcUpdErr := svc.Update(); svcUpdErr != nil {
		if apierrors.IsNotFound(svcUpdErr) {
			if createErr := svc.Create(); createErr != nil {
				return createErr
			}
		} else {
			r.Log.Info("failed to update jitsi service", "error", svcUpdErr)
			return svcUpdErr
		}
	}
	return nil
}

func (r *Reconciler) makeJVB(ctx context.Context, j *v1alpha1.Jitsi) {
	if j.Status.JVBReplicas > j.Spec.JVB.Replicas {
		replica := j.Status.JVBReplicas
		delta := j.Status.JVBReplicas - j.Spec.JVB.Replicas
		for replica >= delta && replica != 1 {
			jts := jitsi.NewJVB(ctx, replica, j, r.Client, r.Log)
			if delErr := jts.Delete(); delErr != nil {
				r.Log.Info("can't scale down jvb replicas", "error", delErr)
			}
			replica--
		}
	}
	for replica := int32(1); replica <= j.Spec.JVB.Replicas; replica++ {
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
	j.Status.JVBReplicas = j.Spec.JVB.Replicas
	if err := r.Client.Status().Update(ctx, j); err != nil {
		r.Log.Info("can't update jitsi custom resource status", "error", err)
	}
}

func (r *Reconciler) onDelete(ctx context.Context, j *v1alpha1.Jitsi) {
	for _, appName := range jitsiServices {
		if err := r.deleteComponents(ctx, appName, j); err != nil {
			r.Log.Info("failed to delete component", "error", err)
		}
	}
	r.deleteJVB(ctx, j)
	if utils.ContainsString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		j.ObjectMeta.Finalizers = utils.RemoveString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer)
		if err := r.Client.Update(ctx, j); err != nil {
			r.Log.Info("can't update jitsi cr", "error", err)
		}
	}
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
		jvb := jitsi.NewJVB(ctx, replica, j, r.Client, r.Log)
		if err := jvb.Delete(); client.IgnoreNotFound(err) != nil {
			r.Log.Info("failed to delete jvb instance", "error", err)
		}
	}
}
