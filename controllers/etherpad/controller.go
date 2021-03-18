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
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	etherpadv1alpha1 "github.com/onmetal/meeting-operator/api/v1alpha1"
)

type EtherpadReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	appKubernetesName   = "etherpad"
	appKubernetesPartOf = "jitsi"
	etherpadFinalizer   = "etherpad.finalizers.meeting.io"
)

//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etherpad.meeting.ko,resources=etherpads/finalizers,verbs=update
func (r *EtherpadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("etherpad", req.NamespacedName)

	etherpad := &etherpadv1alpha1.Etherpad{}
	if err := r.Get(ctx, req.NamespacedName, etherpad); err != nil {
		//r.Log.Error(err, "unable to fetch Etherpad")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if etherpad.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(etherpad.ObjectMeta.Finalizers, etherpadFinalizer) {
			etherpad.ObjectMeta.Finalizers = append(etherpad.ObjectMeta.Finalizers, etherpadFinalizer)
			if err := r.Update(context.Background(), etherpad); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(etherpad.ObjectMeta.Finalizers, etherpadFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(etherpad); err != nil {
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			etherpad.ObjectMeta.Finalizers = removeString(etherpad.ObjectMeta.Finalizers, etherpadFinalizer)
			if err := r.Update(context.Background(), etherpad); err != nil {
				return ctrl.Result{}, err
			}
			r.Log.Info("external resources were deleted")
		}
		return ctrl.Result{}, nil
	}
	deployment, err := r.getDeployment(ctx, etherpad.Name, etherpad.Namespace)
	if err != nil && errors.IsNotFound(err) {
		if err := r.createDeployment(ctx, etherpad); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updateDeployment(ctx, deployment, etherpad); err != nil {
			return ctrl.Result{}, err
		}
		r.Log.Info("deployment updated:", "name", deployment.Name)
	}
	service, err := r.getService(ctx, etherpad.Name, etherpad.Namespace)
	if err != nil && errors.IsNotFound(err) {
		if err := r.createService(ctx, etherpad); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updateService(ctx, service, etherpad); err != nil {
			return ctrl.Result{}, err
		}
		r.Log.Info("service updated:", "name", service.Name)
	}

	return ctrl.Result{}, nil
}

func (r *EtherpadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etherpadv1alpha1.Etherpad{}).
		Owns(&appsv1.Deployment{}).
		Owns(&v1.Service{}).
		Complete(r)
}

func (r *EtherpadReconciler) deleteExternalResources(etherpad *etherpadv1alpha1.Etherpad) error {
	if err := r.deleteDeployment(context.Background(), etherpad); err != nil {
		return err
	}
	if err := r.deleteService(context.Background(), etherpad); err != nil {
		return err
	}
	return nil
}

func getDefaultLabels() map[string]string {
	var defaultLabels = make(map[string]string)

	defaultLabels["app.kubernetes.io/name"] = appKubernetesName
	defaultLabels["app.kubernetes.io/part-of"] = appKubernetesPartOf

	return defaultLabels
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
