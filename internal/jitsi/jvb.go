package jitsi

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	JvbPodName       = "jitsi-jvb"
	JvbContainerName = "jvb"
	externalPort     = 10000
	waitForDeletion  = 40 * time.Second
	waitForLB        = 20 * time.Second
)

func (j *JVB) Create() error {
	if err := j.createServicePerPod(); err != nil {
		j.log.Info("failed to create service", "error", err, "namespace", j.namespace)
	}
	err := j.createPod()
	if err != nil {
		j.log.Info("failed to create pod", "error", err, "namespace", j.namespace)
	}
	return nil
}

func (j *JVB) createServicePerPod() error {
	prepareService := j.prepareServiceForPod()
	err := j.Client.Create(j.ctx, prepareService)
	if errors.IsAlreadyExists(err) {
		j.log.Info("service already exist", "name", j.serviceName)
		return nil
	}
	return err
}

func (j *JVB) prepareServiceForPod() *v1.Service {
	labelKey := fmt.Sprintf("%s-%d", JvbPodName, j.replica)
	labels := map[string]string{"jitsi-jvb": labelKey}
	port := externalPort + j.replica
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        j.serviceName,
			Namespace:   j.namespace,
			Annotations: j.ServiceAnnotations,
		},
		Spec: v1.ServiceSpec{
			Type: j.ServiceType,
			Ports: []v1.ServicePort{
				{
					Name:       "jvb",
					Protocol:   j.Service.Protocol,
					Port:       port,
					TargetPort: intstr.IntOrString{IntVal: port},
				},
			},
			Selector: labels,
		},
	}
}

func (j *JVB) createPod() error {
	labelKey := fmt.Sprintf("%s-%d", JvbPodName, j.replica)
	labels := map[string]string{"jitsi-jvb": labelKey}
	spec := j.createPopSpec()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.podName,
			Namespace: j.namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
	return j.Client.Create(j.ctx, pod)
}

func (j *JVB) createPopSpec() v1.PodSpec {
	port := externalPort + j.replica
	envs := j.additionalEnvironments()
	return v1.PodSpec{
		ImagePullSecrets: j.ImagePullSecrets,
		Containers: []v1.Container{
			{
				Name:  JvbContainerName,
				Image: j.Image,
				Ports: []v1.ContainerPort{
					{
						Name:          "jvb",
						Protocol:      j.Service.Protocol,
						ContainerPort: port,
					},
				},
				Env: envs,
				Resources: v1.ResourceRequirements{
					Requests: j.Resources,
				},
			},
		},
	}
}

func (j *JVB) additionalEnvironments() []v1.EnvVar {
	port := fmt.Sprint(externalPort + j.replica)
	externalIP := j.getExternalIP()
	switch {
	case j.Service.Protocol == v1.ProtocolTCP:
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
			j.Environments = append(j.Environments, additionalEnvs[index])
		}
		return j.Environments
	case j.Service.Protocol == v1.ProtocolUDP:
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
			j.Environments = append(j.Environments, additionalEnvs[index])
		}
		return j.Environments
	default:
		return j.Environments
	}
}

func (j *JVB) getExternalIP() string {
	serviceName := fmt.Sprintf("%s-%d", JvbPodName, j.replica)
	time.Sleep(waitForLB)
	svc := j.getService(serviceName)
	switch {
	case len(svc.Status.LoadBalancer.Ingress) != 0:
		return svc.Status.LoadBalancer.Ingress[0].IP
	case svc.Spec.LoadBalancerIP != "":
		return svc.Spec.LoadBalancerIP
	default:
		return ""
	}
}

func (j *JVB) getService(serviceName string) v1.Service {
	svc := v1.Service{}
	if err := j.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: j.namespace,
		Name:      serviceName,
	}, &svc); err != nil {
		j.log.Error(err, "failed to get service")
	}
	return svc
}

func (j *JVB) Update() error {
	for replica := int32(1); replica <= j.Replicas; replica++ {
		j.replica = replica
		j.podName = fmt.Sprintf("%s-%d", JvbPodName, replica)
		j.serviceName = fmt.Sprintf("jitsi-jvb-%d", replica)
		if !j.isPodExist() {
			if err := j.Create(); err != nil {
				j.log.Info("failed to update jvb", "error", err, "namespace", j.namespace)
			}
		} else {
			// We can't update pod.spec and deletion is required
			if err := j.Delete(); err != nil {
				j.log.Info("failed to update jvb", "error", err, "namespace", j.namespace)
			}
			if err := j.Create(); err != nil {
				j.log.Info("failed to update jvb", "error", err, "namespace", j.namespace)
			}
		}
	}
	return nil
}

func (j *JVB) isPodExist() bool {
	pod := v1.Pod{}
	err := j.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: j.namespace,
		Name:      j.podName,
	}, &pod)
	if err != nil && errors.IsNotFound(err) {
		return false
	}
	return true
}

func (j *JVB) Delete() error {
	for replica := int32(1); replica <= j.Replicas; replica++ {
		podName := fmt.Sprintf("%s-%d", JvbPodName, replica)
		serviceName := fmt.Sprintf("jitsi-jvb-%d", replica)
		if err := j.deletePod(podName); err != nil {
			j.log.Info("failed to delete pod", "error", err, "namespace", j.namespace)
		}
		if err := j.deleteService(serviceName); err != nil {
			j.log.Info("failed to delete service", "error", err, "namespace", j.namespace)
		}
		time.Sleep(waitForDeletion)
	}
	return nil
}

func (j *JVB) deletePod(podName string) error {
	pod := &v1.Pod{}
	err := j.Client.Get(j.ctx, types.NamespacedName{
		Namespace: j.namespace,
		Name:      podName},
		pod)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		j.log.Info("can't get pod by name", "error", err)
		return err
	}
	return j.Client.Delete(j.ctx, pod)
}

func (j *JVB) deleteService(name string) error {
	svc := &v1.Service{}
	err := j.Client.Get(j.ctx, types.NamespacedName{Namespace: j.namespace, Name: name}, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		j.log.Info("can't get svc by name", "error", err)
		return err
	}
	return j.Client.Delete(j.ctx, svc)
}
