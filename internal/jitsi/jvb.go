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
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	JvbName         = "jvb"
	externalPort    = 10000
)

const (
	timeOutSecond   = 300 * time.Second
	tickTimerSecond = 15 * time.Second
)

func NewJVB(ctx context.Context, replica int32,
	j *v1alpha1.Jitsi, c client.Client, l logr.Logger) *JVB {
	name := fmt.Sprintf("%s-%d", JvbName, replica)
	return &JVB{
		Client:      c,
		JVB:         j.Spec.JVB,
		envs:        j.Spec.JVB.Environments,
		ctx:         ctx,
		log:         l,
		name:        name,
		serviceName: name,
		namespace:   j.Namespace,
		replica:     replica,
	}
}

func (j *JVB) Create() error {
	if j.exist() {
		return apierrors.NewAlreadyExists(appsv1.Resource("deployments"), j.name)
	}
	if err := j.servicePerInstance(); err != nil {
		j.log.Info("failed to create service", "error", err, "namespace", j.namespace)
	}
	err := j.createInstance()
	if err != nil {
		j.log.Info("failed to create sts", "error", err, "namespace", j.namespace)
	}
	return nil
}

func (j *JVB) servicePerInstance() error {
	service, err := j.getService()
	preparedService := j.prepareServiceForInstance()
	if j.Exporter.Type == "" {
		if exporterErr := j.Client.Create(j.ctx, j.serviceForExporter()); err != nil {
			j.log.Info("can't create exporter service", "error", exporterErr)
		}
	}
	switch {
	case apierrors.IsNotFound(err):
		return j.Client.Create(j.ctx, preparedService)
	case service.Spec.Type != j.ServiceType || service.Spec.Ports[0].Protocol != j.Port.Protocol:
		// You can't change spec.type on existing service
		if delErr := j.Client.Delete(j.ctx, service); delErr != nil {
			j.log.Error(delErr, "failed to update service type", "error", delErr)
		}
		if j.isDeleted() {
			return j.Client.Create(j.ctx, preparedService)
		}
		return errors.New("service deletion timeout")
	case service.Spec.Type == v1.ServiceTypeLoadBalancer:
		// can't change serviceAnnotations when service type is LoadBalancer
		if isAnnotationsChanged(service.Annotations, j.ServiceAnnotations) {
			if delErr := j.Client.Delete(j.ctx, service); delErr != nil {
				j.log.Error(delErr, "failed to delete service", "error", delErr)
			}
			if j.isDeleted() {
				return j.Client.Create(j.ctx, preparedService)
			}
			return errors.New("service deletion timeout")
		}
		service.Spec.Ports = preparedService.Spec.Ports
		return j.Client.Update(j.ctx, service)
	default:
		service.Spec.Ports = preparedService.Spec.Ports
		return j.Client.Update(j.ctx, service)
	}
}

func (j *JVB) prepareServiceForInstance() *v1.Service {
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
					Name:       JvbName,
					Protocol:   j.Port.Protocol,
					Port:       port,
					TargetPort: intstr.IntOrString{IntVal: port},
				},
			},
			Selector: map[string]string{"jitsi-jvb": j.name},
		},
	}
}

func (j *JVB) serviceForExporter() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("exporter-%s", j.serviceName),
			Namespace: j.namespace,
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Ports: []v1.ServicePort{
				{
					Name:       "exporter",
					Protocol:   v1.ProtocolTCP,
					Port:       j.Exporter.Port,
					TargetPort: intstr.IntOrString{IntVal: j.Exporter.Port},
				},
			},
			Selector: map[string]string{"jitsi-jvb": j.name},
		},
	}
}

func (j *JVB) isDeleted() bool {
	timeout := time.After(timeOutSecond)
	tick := time.NewTicker(tickTimerSecond)
	for {
		select {
		case <-timeout:
			return false
		case <-tick.C:
			if _, getErr := j.getService(); apierrors.IsNotFound(getErr) {
				return true
			}
		}
	}
}

func (j *JVB) createInstance() error {
	prepared := j.prepareInstance()
	return j.Client.Create(j.ctx, prepared)
}

func (j *JVB) prepareInstance() *appsv1.Deployment {
	labels := map[string]string{"jitsi-jvb": j.name}
	spec := j.prepareDeploymentSpec(labels)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.name,
			Namespace: j.namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
}

func (j *JVB) prepareDeploymentSpec(labels map[string]string) appsv1.DeploymentSpec {
	jvb := j.prepareJVBContainer()
	exporter, volume := j.prepareExporterContainer()
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: v1.PodSpec{
				ImagePullSecrets: j.ImagePullSecrets,
				Volumes:          volume,
				Containers: []v1.Container{
					jvb,
					exporter,
				},
			},
		},
	}
}

func (j *JVB) prepareJVBContainer() v1.Container {
	port := externalPort + j.replica
	return v1.Container{
		Name:            JvbName,
		Image:           j.Image,
		ImagePullPolicy: j.ImagePullPolicy,
		Env:             j.additionalEnvironments(),
		Resources:       j.Resources,
		Ports: []v1.ContainerPort{
			{
				Name:          JvbName,
				Protocol:      j.Port.Protocol,
				ContainerPort: port,
			},
		},
	}
}

func (j *JVB) additionalEnvironments() []v1.EnvVar {
	port := fmt.Sprint(externalPort + j.replica)
	switch {
	case j.Port.Protocol == v1.ProtocolTCP:
		additionalEnvs := make([]v1.EnvVar, 0, 6)
		if !isHostAddressExist(j.envs) {
			additionalEnvs = append(additionalEnvs, j.getDockerHostAddr())
		}
		additionalEnvs = append(additionalEnvs,
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
			j.envs = append(j.envs, additionalEnvs[index])
		}
		return j.envs
	case j.Port.Protocol == v1.ProtocolUDP:
		additionalEnvs := make([]v1.EnvVar, 0, 2)
		if !isHostAddressExist(j.envs) {
			additionalEnvs = append(additionalEnvs, j.getDockerHostAddr())
		}
		additionalEnvs = append(additionalEnvs,
			v1.EnvVar{
				Name:  "JVB_PORT",
				Value: port,
			})
		for env := range additionalEnvs {
			j.envs = append(j.envs, additionalEnvs[env])
		}
		return j.envs
	default:
		return j.envs
	}
}

func (j *JVB) getDockerHostAddr() v1.EnvVar {
	if j.ServiceType != v1.ServiceTypeLoadBalancer {
		return v1.EnvVar{
			Name:  "DOCKER_HOST_ADDRESS",
			Value: "",
		}
	}
	return v1.EnvVar{
		Name:  "DOCKER_HOST_ADDRESS",
		Value: j.getExternalIP(),
	}
}

func (j *JVB) getExternalIP() string {
	timeout := time.After(timeOutSecond)
	tick := time.NewTicker(tickTimerSecond)
	for {
		select {
		case <-timeout:
			return ""
		case <-tick.C:
			svc, err := j.getService()
			if apierrors.IsNotFound(err) {
				j.log.Info("can't get svc by name", "error", err)
				return ""
			}
			if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
				return ""
			}
			if len(svc.Status.LoadBalancer.Ingress) != 0 {
				return svc.Status.LoadBalancer.Ingress[0].IP
			}
			if svc.Spec.LoadBalancerIP != "" {
				return svc.Spec.LoadBalancerIP
			}
		}
	}
}

func (j *JVB) getService() (*v1.Service, error) {
	svc := &v1.Service{}
	if err := j.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: j.namespace,
		Name:      j.serviceName,
	}, svc); err != nil {
		return &v1.Service{}, err
	}
	return svc, nil
}

func (j *JVB) prepareExporterContainer() (v1.Container, []v1.Volume) {
	switch j.Exporter.Type {
	case "telegraf":
		var volume []v1.Volume
		volume = append(volume, v1.Volume{
			Name: "configuration",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: j.Exporter.ConfigMapName},
				},
			},
		})
		return v1.Container{
			Name:            "exporter",
			Image:           j.Exporter.Image,
			Env:             j.Exporter.Environments,
			Resources:       j.Exporter.Resources,
			VolumeMounts:    []v1.VolumeMount{{Name: "configuration", MountPath: "/etc/telegraf/"}},
			ImagePullPolicy: j.Exporter.ImagePullPolicy,
			SecurityContext: &j.Exporter.SecurityContext,
		}, volume
	default:
		return v1.Container{
			Name:            "exporter",
			Image:           j.Exporter.Image,
			Args:            []string{"-videobridge-url", "http://localhost:8080/colibri/stats"},
			Ports:           []v1.ContainerPort{{Name: "http", ContainerPort: j.Exporter.Port, Protocol: v1.ProtocolTCP}},
			Env:             j.Exporter.Environments,
			Resources:       j.Exporter.Resources,
			ImagePullPolicy: j.Exporter.ImagePullPolicy,
			SecurityContext: &j.Exporter.SecurityContext,
		}, nil
	}
}

func (j *JVB) Update() error {
	instance, err := j.getInstance()
	if err != nil {
		j.log.Info("failed to get jvb instance", "error", err)
		return err
	}
	if svcCreationErr := j.servicePerInstance(); svcCreationErr != nil {
		j.log.Info("failed to create service", "error", svcCreationErr, "namespace", j.namespace)
	}
	prepared := j.prepareInstance()
	instance.Spec.Template.Spec = prepared.Spec.Template.Spec
	return j.Client.Update(j.ctx, instance)
}

func (j *JVB) Delete() error {
	if err := j.deleteInstance(); client.IgnoreNotFound(err) != nil {
		j.log.Info("failed to delete instance", "error", err, "namespace", j.namespace)
	}
	if err := j.deleteService(); client.IgnoreNotFound(err) != nil {
		j.log.Info("failed to delete service", "error", err, "namespace", j.namespace)
	}
	return nil
}

func (j *JVB) exist() bool {
	_, err := j.getInstance()
	if err != nil && apierrors.IsNotFound(err) {
		return false
	}
	return true
}

func (j *JVB) getInstance() (*appsv1.Deployment, error) {
	d := &appsv1.Deployment{}
	err := j.Client.Get(context.TODO(), types.NamespacedName{Namespace: j.namespace, Name: j.name}, d)
	return d, err
}

func (j *JVB) deleteInstance() error {
	d := &appsv1.Deployment{}
	err := j.Client.Get(j.ctx, types.NamespacedName{Namespace: j.namespace, Name: j.name}, d)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		j.log.Info("can't get instance by name", "error", err)
		return err
	}
	return j.Client.Delete(j.ctx, d)
}

func (j *JVB) deleteService() error {
	svc, err := j.getService()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		j.log.Info("can't get svc by name", "error", err)
		return err
	}
	if deleteErr := j.Client.Delete(j.ctx, svc); deleteErr != nil {
		return deleteErr
	}
	exporterSvc, err := j.getExporterService()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		j.log.Info("can't get svc by name", "error", err)
		return err
	}
	return j.Client.Delete(j.ctx, exporterSvc)
}

func (j *JVB) getExporterService() (*v1.Service, error) {
	svc := &v1.Service{}
	if err := j.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: j.namespace,
		Name:      fmt.Sprintf("exporter-%s", j.serviceName),
	}, svc); err != nil {
		return &v1.Service{}, err
	}
	return svc, nil
}

func isHostAddressExist(envs []v1.EnvVar) bool {
	for env := range envs {
		if envs[env].Name != "DOCKER_HOST_ADDRESS" {
			continue
		}
		return true
	}
	return false
}
