package jitsi

import (
	"context"
	"fmt"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	JvbPodName       = "jitsi-jvb"
	JvbContainerName = "jvb"
	externalPort     = 10000
	waitForDeletion  = 40 * time.Second
	waitForLB        = 20 * time.Second
)

func (r *Reconciler) makeJVB(ctx context.Context, jitsi *v1alpha1.Jitsi) error {
	for replica := int32(1); replica <= jitsi.Spec.JVB.Replicas; replica++ {
		podName := fmt.Sprintf("%s-%d", JvbPodName, replica)
		serviceName := fmt.Sprintf("jitsi-jvb-%d", replica)
		if err := r.createServicePerPod(ctx, serviceName, jitsi.Namespace, jitsi.Spec.JVB.Protocol,
			replica, jitsi.Spec.JVB.Type); err != nil {
			r.Log.Info("failed to create service", "error", err, "namespace", jitsi.Namespace)
		}
		if r.isPodExist(podName, jitsi.Namespace) {
			if err := r.deletePod(ctx, podName, jitsi.Namespace); err != nil {
				r.Log.Info("failed to delete pod", "error", err, "namespace", jitsi.Namespace)
			}
			time.Sleep(waitForDeletion)
			err := r.createPod(ctx, replica, podName, jitsi.Namespace, &jitsi.Spec.JVB)
			if err != nil {
				r.Log.Info("failed to create pod", "error", err, "namespace", jitsi.Namespace)
				return err
			}
		} else {
			err := r.createPod(ctx, replica, podName, jitsi.Namespace, &jitsi.Spec.JVB)
			if err != nil {
				r.Log.Info("failed to create pod", "error", err, "namespace", jitsi.Namespace)
			}
		}
	}
	return nil
}

func (r *Reconciler) createPod(ctx context.Context, replica int32,
	podName, namespace string, jvb *v1alpha1.JVB) error {
	labelKey := fmt.Sprintf("%s-%d", JvbPodName, replica)
	labels := map[string]string{"jitsi-jvb": labelKey}
	spec := r.createPopSpec(replica, namespace, jvb)
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

func (r *Reconciler) createPopSpec(replica int32, namespace string, jvb *v1alpha1.JVB) v1.PodSpec {
	port := externalPort + replica
	envs := r.additionalEnvironments(replica, namespace, jvb.Service.Protocol, jvb.Environments)
	return v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  JvbContainerName,
				Image: jvb.Image,
				Ports: []v1.ContainerPort{
					{
						Name:          "jvb",
						Protocol:      v1.Protocol(jvb.Service.Protocol),
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

func (r *Reconciler) additionalEnvironments(replica int32, namespace, protocol string, envs []v1.EnvVar) []v1.EnvVar {
	port := fmt.Sprint(externalPort + replica)
	externalIP := r.getExternalIP(namespace, replica)
	switch protocol {
	case "TCP":
		additionalEnvs := make([]v1.EnvVar, 0, 6)
		additionalEnvs = append(additionalEnvs,
			v1.EnvVar{
				Name:  "DOCKER_HOST_ADDRESS",
				Value: externalIP,
			},
			v1.EnvVar{
				Name:  "JVB_PORT",
				Value: "30300",
			},
			v1.EnvVar{
				Name:  "JVB_TCP_PORT",
				Value: port,
			},
			v1.EnvVar{
				Name:  "JVB_TCP_MAPPED_PORT",
				Value: port,
			},
			v1.EnvVar{
				Name:  "TCP_HARVESTER_PORT",
				Value: port,
			},
			v1.EnvVar{
				Name:  "TCP_HARVESTER_MAPPED_PORT",
				Value: port,
			})
		for index := range additionalEnvs {
			envs = append(envs, additionalEnvs[index])
		}
		return envs
	case "UDP":
		additionalEnvs := make([]v1.EnvVar, 0, 2)
		additionalEnvs = append(additionalEnvs,
			v1.EnvVar{
				Name:  "DOCKER_HOST_ADDRESS",
				Value: externalIP,
			},
			v1.EnvVar{
				Name:  "JVB_PORT",
				Value: port,
			})
		for index := range additionalEnvs {
			envs = append(envs, additionalEnvs[index])
		}
		return envs
	default:
		return envs
	}
}

func (r *Reconciler) createServicePerPod(ctx context.Context, name, namespace, protocol string,
	replica int32, serviceType v1.ServiceType) error {
	svc := prepareServiceForPod(name, namespace, protocol, replica, serviceType)
	err := r.Create(ctx, svc)
	if errors.IsAlreadyExists(err) {
		r.Log.Info("service already exist", "name", name)
		return nil
	}
	return err
}

func prepareServiceForPod(name, namespace, protocol string, replica int32, serviceType v1.ServiceType) *v1.Service {
	labelKey := fmt.Sprintf("%s-%d", JvbPodName, replica)
	labels := map[string]string{"jitsi-jvb": labelKey}
	port := externalPort + replica
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Type: serviceType,
			Ports: []v1.ServicePort{
				{
					Name:       "jvb",
					Protocol:   v1.Protocol(protocol),
					Port:       port,
					TargetPort: intstr.IntOrString{IntVal: port},
				},
			},
			Selector: labels,
		},
	}
}

func (r *Reconciler) getExternalIP(namespace string, replica int32) string {
	serviceName := fmt.Sprintf("%s-%d", JvbPodName, replica)
	time.Sleep(waitForLB)
	svc := r.getService(serviceName, namespace)
	if len(svc.Status.LoadBalancer.Ingress) != 0 {
		return svc.Status.LoadBalancer.Ingress[0].IP
	}
	return ""
}

func (r *Reconciler) getService(serviceName, namespace string) v1.Service {
	svc := v1.Service{}
	if err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      serviceName,
	}, &svc); err != nil {
		r.Log.Error(err, "failed to get service")
	}
	return svc
}

func (r *Reconciler) isPodExist(name, namespace string) bool {
	pod := v1.Pod{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, &pod)
	if err != nil && errors.IsNotFound(err) {
		return false
	}
	return true
}

func (r *Reconciler) cleanUpJVBObjects(ctx context.Context, jitsi *v1alpha1.Jitsi) error {
	for replica := int32(1); replica <= jitsi.Spec.JVB.Replicas; replica++ {
		podName := fmt.Sprintf("%s-%d", JvbPodName, replica)
		serviceName := fmt.Sprintf("jitsi-jvb-%d", replica)
		err := r.deletePod(ctx, podName, jitsi.Namespace)
		if err != nil && !errors.IsNotFound(err) {
			r.Log.Info("failed to delete pod", "error", err, "namespace", jitsi.Namespace)
			return err
		}
		err = r.deleteService(ctx, serviceName, jitsi.Namespace)
		if err != nil && !errors.IsNotFound(err) {
			r.Log.Info("failed to delete service", "error", err, "namespace", jitsi.Namespace)
			return err
		}
	}
	return nil
}

func (r *Reconciler) deletePod(ctx context.Context, podName, namespace string) error {
	pod := &v1.Pod{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      podName},
		pod)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		r.Log.Info("can't get pod by name", "error", err)
		return err
	}
	return r.Delete(ctx, pod)
}

func (r *Reconciler) deleteService(ctx context.Context, name, namespace string) error {
	svc := &v1.Service{}
	err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		r.Log.Info("can't get svc by name", "error", err)
		return err
	}
	return r.Delete(ctx, svc)
}
