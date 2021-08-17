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
	"reflect"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/jitsi"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type JVBReconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *JVBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.JVB{}).
		WithEventFilter(r.constructPredicates()).
		Complete(r)
}

func (r *JVBReconciler) constructPredicates() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: onUpdate,
	}
}

func onUpdate(e event.UpdateEvent) bool {
	oldJVBObj, oldOk := e.ObjectOld.(*v1beta1.JVB)
	newJVBObj, newOk := e.ObjectNew.(*v1beta1.JVB)
	if !oldOk || !newOk {
		return false
	}
	return !reflect.DeepEqual(oldJVBObj.Spec, newJVBObj.Spec) || !newJVBObj.DeletionTimestamp.IsZero()
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jvbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jvbs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jvbs/finalizers,verbs=update

func (r *JVBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)
	jvb, err := jitsi.NewJVB(ctx, r.Client, reqLogger, req)
	if err != nil {
		if meeterr.IsUnderDeletion(err) {
			if delErr := jvb.Delete(); delErr != nil {
				reqLogger.Info("deletion failed", "error", delErr)
				return ctrl.Result{}, delErr
			}
			reqLogger.Info("reconciliation finished")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if createErr := jvb.Create(); createErr != nil {
		return ctrl.Result{}, createErr
	}
	if updErr := jvb.Update(); updErr != nil {
		return ctrl.Result{}, updErr
	}
	if updErr := jvb.UpdateStatus(); updErr != nil {
		return ctrl.Result{}, updErr
	}
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}
