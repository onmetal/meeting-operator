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

package jigasi

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type Reconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
}

type Jigasi struct {
	client.Client
	*v1beta1.Jigasi

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Jigasi{}).
		WithEventFilter(r.constructPredicates()).
		Complete(r)
}

func (r *Reconciler) constructPredicates() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: isUpdated,
		DeleteFunc: func(event.DeleteEvent) bool { return false },
	}
}

// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jigasis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jigasis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=jigasis/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)

	jigasi, err := r.newInstance(ctx, reqLogger, req)
	if err != nil {
		if meeterr.IsUnderDeletion(err) {
			reqLogger.V(1).Info("deletion finished")
			return ctrl.Result{}, jigasi.Delete()
		}
		reqLogger.Error(err, "can't create new instance of web")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	deployment, getErr := jigasi.Get()
	if apierrors.IsNotFound(getErr) {
		if createErr := jigasi.Create(); createErr != nil {
			return ctrl.Result{}, createErr
		}
		reqLogger.V(1).Info("reconciliation finished")
		return ctrl.Result{}, nil
	}

	if updErr := jigasi.Update(deployment); updErr != nil {
		return ctrl.Result{}, updErr
	}

	reqLogger.Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) newInstance(ctx context.Context, l logr.Logger, req ctrl.Request) (*Jigasi, error) {
	j := &v1beta1.Jigasi{}
	if err := r.Get(ctx, req.NamespacedName, j); err != nil {
		return nil, err
	}
	defaultLabels := utils.GetDefaultLabelsForApp(name)
	if !j.DeletionTimestamp.IsZero() {
		return &Jigasi{
			Client:    r.Client,
			Jigasi:    j,
			name:      name,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    defaultLabels,
		}, meeterr.UnderDeletion()
	}
	if err := utils.AddFinalizer(ctx, r.Client, j); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &Jigasi{
		Client:    r.Client,
		Jigasi:    j,
		ctx:       ctx,
		log:       l,
		name:      name,
		namespace: j.Namespace,
		labels:    defaultLabels,
	}, nil
}

func isUpdated(e event.UpdateEvent) bool {
	oldObj, oldOk := e.ObjectOld.(*v1beta1.Prosody)
	newObj, newOk := e.ObjectNew.(*v1beta1.Prosody)
	if !oldOk || !newOk {
		return false
	}
	if len(oldObj.Finalizers) < 1 && len(newObj.Finalizers) >= 1 {
		return false
	}
	return true
}
