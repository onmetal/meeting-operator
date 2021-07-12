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
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"k8s.io/apimachinery/pkg/labels"

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
	JvbName      = "jvb"
	externalPort = 10000
)

const (
	timeOutSecond   = 300 * time.Second
	tickTimerSecond = 15 * time.Second
)

const telegrafExporter = "telegraf"

func NewJVB(ctx context.Context, replica int32,
	j *v1alpha1.Jitsi, c client.Client, l logr.Logger) Jitsi {
	name := fmt.Sprintf("%s-%d", JvbName, replica)
	deleted := !j.DeletionTimestamp.IsZero()
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
		deleted:     deleted,
	}
}

func (j *JVB) Create() error {
	if j.exist() {
		return meetingerr.AlreadyExist(j.name)
	}
	if err := j.createShutdownCM(); err != nil {
		j.log.Info("can't create graceful shutdown config map", "error", err)
	}
	if err := j.createCustomSIPCM(); err != nil {
		j.log.Info("can't create custom sip config map", "error", err)
	}
	if err := j.createCustomLoggingCM(); err != nil {
		j.log.Info("can't create jvb logging config map", "error", err)
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

func (j *JVB) createShutdownCM() error {
	shutdown := j.prepareShutdownCM()
	return j.Client.Create(j.ctx, shutdown)
}

func (j *JVB) createCustomSIPCM() error {
	sip := j.prepareSIPCM()
	return j.Client.Create(j.ctx, sip)
}

func (j *JVB) createCustomLoggingCM() error {
	logging := j.prepareLoggingCM()
	return j.Client.Create(j.ctx, logging)
}

func (j *JVB) prepareShutdownCM() *v1.ConfigMap {
	return &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "jvb-graceful-shutdown", Namespace: j.namespace,
		Labels: map[string]string{"app": "jvb"}},
		Data: map[string]string{"graceful_shutdown.sh": jvbGracefulShutdown}}
}

func (j *JVB) prepareSIPCM() *v1.ConfigMap {
	tpl, err := template.New("sip").Parse(jvbCustomSIP)
	if err != nil {
		j.log.Info("can't template sip config", "error", err)
		return nil
	}
	var b bytes.Buffer
	d := SIP{Options: j.CustomSIP}
	if executeErr := tpl.Execute(&b, d); executeErr != nil {
		j.log.Info("can't template sip config", "error", err)
		return nil
	}
	return &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "jvb-custom-sip", Namespace: j.namespace,
		Labels: map[string]string{"app": "jvb"}},
		Data: map[string]string{"custom-sip-communicator.properties": b.String()}}
}

func (j *JVB) prepareLoggingCM() *v1.ConfigMap {
	tpl, err := template.New("log").Parse(jvbCustomLogging)
	if err != nil {
		j.log.Info("can't template logging config", "error", err)
		return nil
	}
	var level = loggingLevelInfo
	for k := range j.envs {
		if j.envs[k].Name != loggingLevel {
			continue
		}
		level = j.envs[k].Value
	}
	var b bytes.Buffer
	if executeErr := tpl.Execute(&b, level); executeErr != nil {
		j.log.Info("can't template logging config", "error", err)
		return nil
	}
	return &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "jvb-custom-logging", Namespace: j.namespace,
		Labels: map[string]string{"app": "jvb"}},
		Data: map[string]string{"custom-logging.properties": b.String()}}
}

func (j *JVB) servicePerInstance() error {
	service, getErr := j.getService()
	preparedService := j.prepareServiceForInstance()
	if j.Exporter.Type == "" {
		if exporterErr := j.Client.Create(j.ctx, j.serviceForExporter()); exporterErr != nil && !apierrors.IsAlreadyExists(exporterErr) {
			j.log.Info("can't create exporter service", "error", exporterErr)
		}
	}
	switch {
	case apierrors.IsNotFound(getErr):
		return j.Client.Create(j.ctx, preparedService)
	default:
		service.ObjectMeta.Annotations = preparedService.Annotations
		service.Spec.Ports = preparedService.Spec.Ports
		service.Spec.Type = j.ServiceType
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
			Type:     j.ServiceType,
			Ports:    []v1.ServicePort{{Name: JvbName, Protocol: j.Port.Protocol, Port: port, TargetPort: intstr.IntOrString{IntVal: port}}},
			Selector: map[string]string{"jitsi-jvb": j.name},
		},
	}
}

func (j *JVB) serviceForExporter() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("exporter-%s", j.serviceName),
			Namespace: j.namespace,
			Labels: map[string]string{
				"app":                   "jvb",
				"kubernetes.io/part-of": "jitsi"}},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Ports: []v1.ServicePort{{Name: "exporter", Protocol: v1.ProtocolTCP,
				Port: j.Exporter.Port, TargetPort: intstr.IntOrString{IntVal: j.Exporter.Port}}},
			Selector: map[string]string{"jitsi-jvb": j.name},
		},
	}
}

func (j *JVB) createInstance() error {
	prepared := j.prepareInstance()
	return j.Client.Create(j.ctx, prepared)
}

func (j *JVB) prepareInstance() *appsv1.Deployment {
	l := map[string]string{"jitsi-jvb": j.name}
	spec := j.prepareDeploymentSpec(l)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        j.name,
			Namespace:   j.namespace,
			Labels:      l,
			Annotations: j.Annotations,
		},
		Spec: spec,
	}
}

func (j *JVB) prepareDeploymentSpec(l map[string]string) appsv1.DeploymentSpec {
	jvb := j.prepareJVBContainer()
	exporter := j.prepareExporterContainer()
	volumes := j.prepareVolumesForJVB()
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: l,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: l,
			},
			Spec: v1.PodSpec{
				ImagePullSecrets: j.ImagePullSecrets,
				Volumes:          volumes,
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
		SecurityContext: &j.SecurityContext,
		VolumeMounts: []v1.VolumeMount{
			{Name: "shutdown", MountPath: "/shutdown"},
			{Name: "custom-sip", MountPath: "/defaults/sip-communicator.properties", SubPath: "sip-communicator.properties"},
			{Name: "custom-logging", MountPath: "/defaults/logging.properties", SubPath: "logging.properties"}},
		Lifecycle: &v1.Lifecycle{
			PreStop: &v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"bash", "/shutdown/graceful_shutdown.sh", "-t 3"},
				},
			},
		},
		LivenessProbe: &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   "/about/health",
					Port:   intstr.IntOrString{IntVal: 8080},
					Scheme: v1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      30,
			PeriodSeconds:       15,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		},
		Ports: []v1.ContainerPort{
			{
				Name:          JvbName,
				Protocol:      j.Port.Protocol,
				ContainerPort: port,
			},
			{
				Name:          "colibri",
				Protocol:      v1.ProtocolTCP,
				ContainerPort: 8080,
			},
		},
	}
}

func (j *JVB) prepareVolumesForJVB() []v1.Volume {
	var volume []v1.Volume
	var permissions int32 = 0744
	shutdown := v1.Volume{Name: "shutdown", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
		DefaultMode: &permissions, LocalObjectReference: v1.LocalObjectReference{Name: "jvb-graceful-shutdown"}}}}
	sipConfig := v1.Volume{Name: "custom-sip", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
		Items:                []v1.KeyToPath{{Key: "custom-sip-communicator.properties", Path: "sip-communicator.properties"}},
		LocalObjectReference: v1.LocalObjectReference{Name: "jvb-custom-sip"}}}}
	loggingConfig := v1.Volume{Name: "custom-logging", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
		Items:                []v1.KeyToPath{{Key: "custom-logging.properties", Path: "logging.properties"}},
		LocalObjectReference: v1.LocalObjectReference{Name: "jvb-custom-logging"}}}}
	if j.Exporter.Type == telegrafExporter {
		telegrafCM := v1.Volume{Name: "telegraf", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
			LocalObjectReference: v1.LocalObjectReference{Name: j.Exporter.ConfigMapName}}}}
		return append(volume, shutdown, sipConfig, telegrafCM, loggingConfig)
	}
	return append(volume, shutdown, sipConfig, loggingConfig)
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

func (j *JVB) prepareExporterContainer() v1.Container {
	switch j.Exporter.Type {
	case "telegraf":
		return v1.Container{
			Name:            "exporter",
			Image:           j.Exporter.Image,
			Env:             j.Exporter.Environments,
			Resources:       j.Exporter.Resources,
			VolumeMounts:    []v1.VolumeMount{{Name: "telegraf", MountPath: "/etc/telegraf/"}},
			ImagePullPolicy: j.Exporter.ImagePullPolicy,
			SecurityContext: &j.Exporter.SecurityContext,
		}
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
		}
	}
}

func (j *JVB) Update() error {
	if err := j.updateShutdownCM(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := j.createShutdownCM(); createErr != nil {
				j.log.Info("can't create jvb shutdown cm", "error", createErr)
			}
		} else { j.log.Info("can't update jvb shutdown cm", "error", err) }
	}
	if err := j.updateCustomSIPCM(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := j.createCustomSIPCM(); createErr != nil {
				j.log.Info("can't create jvb sip cm", "error", createErr)
			}
		} else { j.log.Info("can't update jvb sip cm", "error", err) }

	}
	if err := j.updateCustomLoggingCM(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := j.createCustomLoggingCM(); createErr != nil {
				j.log.Info("can't create jvb logging cm", "error", createErr)
			}
		} else { j.log.Info("can't update jvb logging cm", "error", err) }
	}
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

func (j *JVB) updateShutdownCM() error {
	s := j.prepareShutdownCM()
	return j.Client.Update(j.ctx, s)
}

func (j *JVB) updateCustomSIPCM() error {
	sip := j.prepareSIPCM()
	return j.Client.Update(j.ctx, sip)
}

func (j *JVB) updateCustomLoggingCM() error {
	logging := j.prepareLoggingCM()
	return j.Client.Update(j.ctx, logging)
}

func (j *JVB) Delete() error {
	if err := j.deleteInstance(); client.IgnoreNotFound(err) != nil {
		j.log.Info("failed to delete instance", "error", err, "namespace", j.namespace)
	}
	if err := j.deleteService(); client.IgnoreNotFound(err) != nil {
		j.log.Info("failed to delete service", "error", err, "namespace", j.namespace)
	}
	if j.deleted {
		if err := j.deleteCMs(); client.IgnoreNotFound(err) != nil {
			j.log.Info("failed to delete jvb cm", "error", err, "namespace", j.namespace)
		}
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

func (j *JVB) deleteCMs() error {
	var cms v1.ConfigMapList
	filter := &client.ListOptions{
		LabelSelector: client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(map[string]string{"app": "jvb"})}}

	if err := j.Client.List(j.ctx, &cms, filter); err != nil {
		return err
	}
	for cm := range cms.Items {
		if err := j.Client.Delete(j.ctx, &cms.Items[cm]); err != nil {
			j.log.Info("can't delete config map", "name", cms.Items[cm].Name, "error", err)
		}
	}
	return nil
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
