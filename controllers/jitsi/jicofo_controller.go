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

	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/jitsi"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type JicofoReconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *JicofoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Jicofo{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jicofos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jicofos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jicofos/finalizers,verbs=update

func (r *JicofoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)
	jicofo, err := jitsi.NewJicofo(ctx, r.Client, reqLogger, req)
	if err != nil {
		if meeterr.IsUnderDeletion(err) {
			if delErr := jicofo.Delete(); delErr != nil {
				reqLogger.Info("deletion failed")
				return ctrl.Result{}, delErr
			}
		}
		reqLogger.Error(err, "can't create new instance of jicofo")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if createErr := jicofo.Create(); createErr != nil {
		if apierrors.IsAlreadyExists(createErr) {
			if updErr := jicofo.Update(); updErr != nil {
				return ctrl.Result{}, updErr
			}
			reqLogger.Info("reconciliation finished")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, createErr
	}
	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}
