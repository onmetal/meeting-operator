package etherpad

import (
	"context"
	etherpadv1alpha1 "github.com/onmetal/meeting-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *EtherpadReconciler) createService(ctx context.Context, etherpad *etherpadv1alpha1.Etherpad) error {
	preparedService := prepareService(etherpad)
	return r.Create(ctx, preparedService)
}

func (r *EtherpadReconciler) updateService(ctx context.Context, service *v1.Service,
	etherpad *etherpadv1alpha1.Etherpad) error {

	switch {
	case service.Spec.Type != etherpad.Spec.Type:
		// You can't change spec.type on existing service
		if err := r.Delete(ctx, service); err != nil {
			r.Log.Error(err, "failed to delete service")
		}
		preparedService := prepareService(etherpad)
		return r.Create(ctx, preparedService)
	default:
		return nil
	}

}
func prepareService(etherpad *etherpadv1alpha1.Etherpad) *v1.Service {
	defaultLabels := getDefaultLabels()
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etherpad.Name,
			Namespace: etherpad.Namespace,
		},
		Spec: v1.ServiceSpec{
			Type: etherpad.Spec.Type,
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					TargetPort: intstr.IntOrString{IntVal: etherpad.Spec.ContainerPort},
					Port:       9001,
					Protocol:   "TCP",
				},
			},
			Selector: defaultLabels,
		},
	}
}

func (r *EtherpadReconciler) getService(ctx context.Context, name, namespace string) (*v1.Service, error) {
	service := &v1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, service)
	return service, err
}

func (r *EtherpadReconciler) deleteService(ctx context.Context, etherpad *etherpadv1alpha1.Etherpad) error {
	service, err := r.getService(ctx, etherpad.Name, etherpad.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("service not found", "name", service.Name)
			return nil
		}
		r.Log.Error(err, "failed to get service")
		return nil
	}
	return r.Delete(ctx, service)
}