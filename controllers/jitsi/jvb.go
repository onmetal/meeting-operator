package jitsi

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"time"

	jitsiv1alpha1 "github.com/onmetal/meeting-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	JvbPodName       = "jitsi-jvb"
	JvbContainerName = "jvb"
	externalPort     = 10000
	waitForDeletion  = 45
)

func (r *Reconciler) makeJVB(ctx context.Context, jitsi *jitsiv1alpha1.Jitsi) error {
	for replica := int32(1); replica <= jitsi.Spec.JVB.Replicas; replica++ {
		err := r.createPod(ctx, replica, jitsi.Namespace, &jitsi.Spec.JVB)
		if err != nil && errors.IsAlreadyExists(err) {
			if err := r.deletePod(ctx, replica, jitsi.Namespace); err != nil {
				r.Log.Info("failed to delete pod", "error", err, "namespace", jitsi.Namespace)
			}
			time.Sleep(waitForDeletion * time.Second)
			err := r.createPod(ctx, replica, jitsi.Namespace, &jitsi.Spec.JVB)
			if err != nil {
				r.Log.Info("failed to create pod", "error", err, "namespace", jitsi.Namespace)
				return err
			}
		}
		serviceName := fmt.Sprintf("jitsi-jvb-%d", replica)
		if err := r.createServicePerPod(ctx, serviceName, jitsi.Namespace, replica); err != nil {
			r.Log.Info("failed to create service", "error", err, "namespace", jitsi.Namespace)
		}
	}
	return nil
}

func (r *Reconciler) createPod(ctx context.Context, replica int32, namespace string, jvb *jitsiv1alpha1.JVB) error {
	spec := r.createPopSpec(replica, jvb)
	podName := fmt.Sprintf("%s-%d", JvbPodName, replica)
	labelKey := fmt.Sprintf("%s-%d", JvbPodName, replica)
	labels := map[string]string{"jitsi-jvb": labelKey}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
	return r.Create(ctx, pod)
}
func (r *Reconciler) deletePod(ctx context.Context, replica int32, namespace string) error {
	podName := fmt.Sprintf("%s-%d", JvbPodName, replica)
	pod := &v1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: podName}, pod)
	if err != nil {
		r.Log.Info("can't get pod by name", "error", err)
		return err
	}
	return r.Delete(ctx, pod)
}

func (r *Reconciler) deleteService(ctx context.Context, name, namespace string) error {
	svc := &v1.Service{}
	err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, svc)
	if err != nil {
		r.Log.Info("can't get svc by name", "error", err)
		return err
	}
	return r.Delete(ctx, svc)
}

func (r *Reconciler) createServicePerPod(ctx context.Context, name, namespace string, replica int32) error {
	svc := prepareServiceForPod(name, namespace, replica)
	err := r.Create(ctx, svc)
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (r *Reconciler) additionalEnvironments(envs []v1.EnvVar, replica int32) []v1.EnvVar {
	jvbTCPPort := fmt.Sprint(externalPort + replica)
	additionalEnvs := []v1.EnvVar{
		{
			Name:  "JVB_TCP_PORT",
			Value: jvbTCPPort,
		},
		{
			Name:  "JVB_TCP_MAPPED_PORT",
			Value: jvbTCPPort,
		},
		{
			Name:  "TCP_HARVESTER_PORT",
			Value: jvbTCPPort,
		},
		{
			Name:  "TCP_HARVESTER_MAPPED_PORT",
			Value: jvbTCPPort,
		},
	}
	for index := range additionalEnvs {
		envs = append(envs, additionalEnvs[index])
	}
	return envs
}

func (r *Reconciler) createPopSpec(replica int32, jvb *jitsiv1alpha1.JVB) v1.PodSpec {
	port := externalPort + replica
	envs := r.additionalEnvironments(jvb.Environments, replica)

	return v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  JvbContainerName,
				Image: jvb.Image,
				Ports: []v1.ContainerPort{
					{
						Name:          "jvb",
						Protocol:      "TCP",
						ContainerPort: port,
					},
				},
				Env: envs,
				Resources: v1.ResourceRequirements{
					Requests: jvb.Resources,
				},
			},
		},
	}
}

func prepareServiceForPod(name, namespace string, replica int32) *v1.Service {
	labelKey := fmt.Sprintf("%s-%d", JvbPodName, replica)
	labels := map[string]string{"jitsi-jvb": labelKey}
	port := externalPort + replica
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "jvb",
					Protocol:   "TCP",
					Port:       port,
					TargetPort: intstr.IntOrString{IntVal: port},
				},
			},
			Selector: labels,
		},
	}
}

func (r *Reconciler) cleanUpJVBObjects(ctx context.Context, jitsi *jitsiv1alpha1.Jitsi) error {
	for replica := int32(1); replica <= jitsi.Spec.JVB.Replicas; replica++ {
		err := r.deletePod(ctx, replica, jitsi.Namespace)
		if err != nil && !errors.IsNotFound(err) {
			r.Log.Info("failed to delete pod", "error", err, "namespace", jitsi.Namespace)
			return err
		}
		serviceName := fmt.Sprintf("jitsi-jvb-%d", replica)
		err = r.deleteService(ctx, serviceName, jitsi.Namespace)
		if err != nil && !errors.IsNotFound(err) {
			r.Log.Info("failed to delete service", "error", err, "namespace", jitsi.Namespace)
			return err
		}
	}
	return nil
}
