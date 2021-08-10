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

type WebReconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *WebReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Web{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=webs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configmaps,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=webs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=webs/finalizers,verbs=update

func (r *WebReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("jitsi", req.NamespacedName, "component", "web")
	web, err := jitsi.NewWeb(ctx, r.Client, reqLogger, req)
	if err != nil {
		if meeterr.IsUnderDeletion(err) {
			if delErr := web.Delete(); delErr != nil {
				reqLogger.Info("deletion failed")
				return ctrl.Result{}, delErr
			}
		}
		reqLogger.Error(err, "can't create new instance of web")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if createErr := web.Create(); createErr != nil {
		if apierrors.IsAlreadyExists(createErr) {
			if updErr := web.Update(); updErr != nil {
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

//func (r *WebReconciler) make(ctx context.Context, j *v1beta1.Web) error {
//	jts, err := jitsi.New(ctx, appName, j, r.Client, r.Log)
//	if err != nil {
//		return err
//	}
//	if jtsUpdErr := jts.Update(); jtsUpdErr != nil {
//		if apierrors.IsNotFound(jtsUpdErr) {
//			if createErr := jts.Create(); createErr != nil {
//				return createErr
//			}
//		} else {
//			r.Log.Info("failed to update jitsi deployment", "error", jtsUpdErr)
//			return jtsUpdErr
//		}
//	}
//	svc, err := jitsi.NewService(ctx, appName, j, r.Client, r.Log)
//	if meeterr.IsNotRequired(err) {
//		return nil
//	}
//	if svcUpdErr := svc.Update(); svcUpdErr != nil {
//		if apierrors.IsNotFound(svcUpdErr) {
//			if createErr := svc.Create(); createErr != nil {
//				return createErr
//			}
//		} else {
//			r.Log.Info("failed to update jitsi service", "error", svcUpdErr)
//			return svcUpdErr
//		}
//	}
//	return nil
//}

//func (r *WebReconciler) onDelete(ctx context.Context, j *v1beta1.Web) {
//	for _, appName := range jitsiServices {
//		if err := r.deleteComponents(ctx, appName, j); err != nil {
//			r.Log.Info("failed to delete component", "error", err)
//		}
//	}
//	if utils.ContainsString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
//		j.ObjectMeta.Finalizers = utils.RemoveString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer)
//		if err := r.Client.Update(ctx, j); err != nil {
//			r.Log.Info("can't update jitsi cr", "error", err)
//		}
//	}
//}

//func (r *WebReconciler) deleteComponents(ctx context.Context, j *v1beta1.Web) error {
//	jts, err := jitsi.NewWeb(ctx, j, r.Client, r.Log)
//	if err != nil {
//		return err
//	}
//	if delErr := jts.Delete(); delErr != nil {
//		return delErr
//	}
//	svc, err := jitsi.NewService(ctx, j, r.Client, r.Log)
//	if meeterr.IsNotRequired(err) {
//		return nil
//	}
//	if delErr := svc.Delete(); delErr != nil {
//		return delErr
//	}
//	return nil
//}
