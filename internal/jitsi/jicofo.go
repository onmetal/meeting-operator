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
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (j *Jicofo) Create() error {
	preparedDeployment := j.prepareDeployment()
	return j.Client.Create(j.ctx, preparedDeployment)
}

func (j *Jicofo) prepareDeployment() *appsv1.Deployment {
	spec := j.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        j.name,
			Namespace:   j.namespace,
			Labels:      j.labels,
			Annotations: j.Annotations,
		},
		Spec: spec,
	}
}

func (j *Jicofo) prepareDeploymentSpec() appsv1.DeploymentSpec {
	volumes := j.prepareVolumesForJicofo()
	jicofo := j.prepareJicofoContainer()
	exporter := j.prepareExporterContainer()
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: j.labels,
		},
		Replicas: &j.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: j.labels,
			},
			Spec: v1.PodSpec{
				ImagePullSecrets: j.ImagePullSecrets,
				Volumes: volumes,
				Containers: []v1.Container{
					jicofo,
					exporter,
				},
			},
		},
	}
}

func (j *Jicofo) prepareVolumesForJicofo() []v1.Volume {
	var volume []v1.Volume
	if j.Exporter.Type == telegrafExporter {
		telegrafCM := v1.Volume{Name: "telegraf", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
			LocalObjectReference: v1.LocalObjectReference{Name: j.Exporter.ConfigMapName}}}}
		return append(volume, telegrafCM)
	}
	return volume
}

func (j *Jicofo) prepareJicofoContainer() v1.Container {
	return v1.Container{
		Name:            JicofoName,
		Image:           j.Image,
		ImagePullPolicy: j.ImagePullPolicy,
		Env:             j.Environments,
		Ports:           getContainerPorts(j.Ports),
		Resources:       j.Resources,
		SecurityContext: &j.SecurityContext,
		LivenessProbe: &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   "/about/health",
					Port:   intstr.IntOrString{IntVal: 8888},
					Scheme: v1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      30,
			PeriodSeconds:       15,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		},
	}
}

func (j *Jicofo) prepareExporterContainer() v1.Container {
	switch j.Exporter.Type {
	case telegrafExporter:
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

func (j *Jicofo) Update() error {
	updatedDeployment := j.prepareDeployment()
	return j.Client.Update(j.ctx, updatedDeployment)
}

func (j *Jicofo) Delete() error {
	deployment, err := j.Get()
	if err != nil {
		return err
	}
	return j.Client.Delete(j.ctx, deployment)
}

func (j *Jicofo) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := j.Client.Get(j.ctx, types.NamespacedName{
		Namespace: j.namespace,
		Name:      j.name,
	}, deployment)
	return deployment, err
}
