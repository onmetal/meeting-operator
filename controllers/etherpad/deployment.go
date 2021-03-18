package etherpad

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"

	etherpadv1alpha1 "github.com/onmetal/meeting-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *EtherpadReconciler) createDeployment(ctx context.Context, etherpad *etherpadv1alpha1.Etherpad) error {
	preparedDeployment := prepareDeployment(etherpad)
	return r.Create(ctx, preparedDeployment)
}

func (r *EtherpadReconciler) updateDeployment(ctx context.Context, deployment *appsv1.Deployment,
	etherpad *etherpadv1alpha1.Etherpad) error {
	updatedSpec := prepareDeploymentSpec(&etherpad.Spec)
	deployment.Spec = updatedSpec
	return r.Update(ctx, deployment)
}

func prepareDeployment(etherpad *etherpadv1alpha1.Etherpad) *appsv1.Deployment {
	spec := prepareDeploymentSpec(&etherpad.Spec)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etherpad.Name,
			Namespace: etherpad.Namespace,
			Labels:    etherpad.Spec.Labels,
		},
		Spec: spec,
	}
}

func prepareDeploymentSpec(etherpadSpec *etherpadv1alpha1.EtherpadSpec) appsv1.DeploymentSpec {
	defaultLabels := getDefaultLabels()
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: defaultLabels,
		},
		Replicas: &etherpadSpec.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: defaultLabels,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "etherpad",
						Image:           etherpadSpec.Image,
						ImagePullPolicy: etherpadSpec.ImagePullPolicy,
						Env:             etherpadSpec.Environments,
						Ports: []v1.ContainerPort{
							{
								Name:          "http",
								ContainerPort: etherpadSpec.ContainerPort,
								Protocol:      "TCP",
							},
						},
					},
				},
			},
		},
	}
}

func (r *EtherpadReconciler) getDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, deployment)
	return deployment, err
}

func (r *EtherpadReconciler) deleteDeployment(ctx context.Context, etherpad *etherpadv1alpha1.Etherpad) error {
	deployment, err := r.getDeployment(ctx, etherpad.Name, etherpad.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("deployment not found", "name", deployment.Name)
		}
		r.Log.Error(err, "failed to get deployment")
	}
	return r.Delete(ctx, deployment)
}