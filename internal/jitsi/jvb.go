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

package jitsi

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	JvbPodName      = "jitsi-jvb"
	JvbName         = "jvb"
	externalPort    = 10000
	timeOutSecond   = 600 * time.Second
	tickTimerSecond = 10 * time.Second
)

func (j *JVB) Create() error {
	if err := j.createServicePerPod(); err != nil {
		j.log.Info("failed to create service", "error", err, "namespace", j.namespace)
	}
	err := j.createSTS()
	if err != nil {
		j.log.Info("failed to create sts", "error", err, "namespace", j.namespace)
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

func (j *JVB) createSTS() error {
	preparedSTS := j.prepareSTS()
	return j.Client.Create(j.ctx, preparedSTS)
}

func (j *JVB) prepareSTS() *appsv1.StatefulSet {
	labelKey := fmt.Sprintf("%s-%d", JvbPodName, j.replica)
	labels := map[string]string{"jitsi-jvb": labelKey}
	spec := j.prepareSTSSpec(labels)
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      labelKey,
			Namespace: j.namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
}

func (j *JVB) prepareSTSSpec(labels map[string]string) appsv1.StatefulSetSpec {
	port := externalPort + j.replica
	envs := j.additionalEnvironments()
	sts := appsv1.StatefulSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		VolumeClaimTemplates: nil,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: v1.PodSpec{
				ImagePullSecrets: j.ImagePullSecrets,
				Containers: []v1.Container{
					{
						Name:            JvbName,
						Image:           j.Image,
						ImagePullPolicy: j.ImagePullPolicy,
						Env:             envs,
						Ports: []v1.ContainerPort{
							{
								Name:          "jvb",
								Protocol:      j.Service.Protocol,
								ContainerPort: port,
							},
						},
					},
				},
			},
		},
	}
	return sts
}

func (j *JVB) additionalEnvironments() []v1.EnvVar {
	port := fmt.Sprint(externalPort + j.replica)
	dockerHostAddr := j.getDockerHostAddr()
	switch {
	case j.Service.Protocol == v1.ProtocolTCP:
		additionalEnvs := make([]v1.EnvVar, 0, 6)
		additionalEnvs = append(additionalEnvs,
			dockerHostAddr,
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
			dockerHostAddr,
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
func (j *JVB) getDockerHostAddr() v1.EnvVar {
	for env := range j.Environments {
		if j.Environments[env].Name != "DOCKER_HOST_ADDRESS" {
			continue
		}
		if j.Environments[env].ValueFrom != nil {
			return v1.EnvVar{
				Name:      j.Environments[env].Name,
				ValueFrom: j.Environments[env].ValueFrom,
			}
		}
		return v1.EnvVar{
			Name:  j.Environments[env].Name,
			Value: j.Environments[env].Value,
		}
	}
	return v1.EnvVar{
		Name:  "DOCKER_HOST_ADDRESS",
		Value: j.getExternalIP(),
	}
}

func (j *JVB) getExternalIP() string {
	serviceName := fmt.Sprintf("%s-%d", JvbPodName, j.replica)
	timeout := time.After(timeOutSecond)
	tick := time.NewTicker(tickTimerSecond)
	for {
		select {
		case <-timeout:
			return ""
		case <-tick.C:
			svc := j.getService(serviceName)
			if len(svc.Status.LoadBalancer.Ingress) != 0 {
				return svc.Status.LoadBalancer.Ingress[0].IP
			}
			if svc.Spec.LoadBalancerIP != "" {
				return svc.Spec.LoadBalancerIP
			}
		}
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
		if !j.isSTSExist(j.podName) {
			if err := j.Create(); err != nil {
				j.log.Info("failed to update jvb", "error", err, "namespace", j.namespace)
			}
		} else {
			// We can't update pod.spec and deletion is required
			if err := j.deleteSTS(j.podName); err != nil {
				j.log.Info("failed to delete pod", "error", err, "namespace", j.namespace)
			}
			if err := j.Create(); err != nil {
				j.log.Info("failed to update jvb", "error", err, "namespace", j.namespace)
			}
		}
	}
	return nil
}

func (j *JVB) Delete() error {
	for replica := int32(1); replica <= j.Replicas; replica++ {
		podName := fmt.Sprintf("%s-%d", JvbPodName, replica)
		serviceName := fmt.Sprintf("jitsi-jvb-%d", replica)
		if err := j.deleteSTS(podName); err != nil {
			j.log.Info("failed to delete pod", "error", err, "namespace", j.namespace)
		}
		if err := j.deleteService(serviceName); err != nil {
			j.log.Info("failed to delete service", "error", err, "namespace", j.namespace)
		}
	}
	return nil
}

func (j *JVB) isSTSExist(name string) bool {
	sts := appsv1.StatefulSet{}
	err := j.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: j.namespace,
		Name:      name,
	}, &sts)
	if err != nil && errors.IsNotFound(err) {
		return false
	}
	return true
}

func (j *JVB) deleteSTS(podName string) error {
	sts := &appsv1.StatefulSet{}
	err := j.Client.Get(j.ctx, types.NamespacedName{
		Namespace: j.namespace,
		Name:      podName},
		sts)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		j.log.Info("can't get pod by name", "error", err)
		return err
	}
	return j.Client.Delete(j.ctx, sts)
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
