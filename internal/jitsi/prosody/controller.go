// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package prosody

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/jitsi"
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

type Prosody struct {
	client.Client
	*v1beta1.Prosody

	Service         jitsi.Servicer
	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Prosody{}).
		WithEventFilter(r.constructPredicates()).
		Complete(r)
}

func (r *Reconciler) constructPredicates() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: isSpecUpdated,
		DeleteFunc: func(event.DeleteEvent) bool { return false },
	}
}

// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=prosodies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=prosodies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jitsi.meeting.ko,resources=prosodies/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("appName", req.Name, "namespace", req.Namespace)

	prosody, err := r.newInstance(ctx, reqLogger, req)
	if err != nil {
		if meeterr.IsUnderDeletion(err) {
			reqLogger.V(1).Info("deletion finished")
			return ctrl.Result{}, prosody.Delete()
		}
		reqLogger.Error(err, "can't create new instance of prosody")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	deployment, getErr := prosody.Get()
	if apierrors.IsNotFound(getErr) {
		if createErr := prosody.Create(); createErr != nil {
			return ctrl.Result{}, createErr
		}
		reqLogger.V(1).Info("reconciliation finished")
		return ctrl.Result{}, nil
	}

	if updErr := prosody.Update(deployment); updErr != nil {
		return ctrl.Result{}, updErr
	}

	reqLogger.V(1).Info("reconciliation finished")
	return ctrl.Result{}, nil
}

func (r *Reconciler) newInstance(ctx context.Context, l logr.Logger, req ctrl.Request) (*Prosody, error) {
	p := &v1beta1.Prosody{}
	if err := r.Get(ctx, req.NamespacedName, p); err != nil {
		return nil, err
	}
	defaultLabels := utils.GetDefaultLabelsForApp(appName)
	s := jitsi.NewService(ctx, r.Client, l, appName, p.Namespace, p.Spec.ServiceAnnotations, defaultLabels, p.Spec.ServiceType, p.Spec.Ports)
	if !p.DeletionTimestamp.IsZero() {
		return &Prosody{
			Client:    r.Client,
			Prosody:   p,
			Service:   s,
			name:      appName,
			namespace: p.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    defaultLabels,
		}, meeterr.UnderDeletion()
	}
	if err := utils.AddFinalizer(ctx, r.Client, p); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &Prosody{
		Client:    r.Client,
		Prosody:   p,
		Service:   s,
		name:      appName,
		namespace: p.Namespace,
		ctx:       ctx,
		log:       l,
		labels:    defaultLabels,
	}, nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
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
