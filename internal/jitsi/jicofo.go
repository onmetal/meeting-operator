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
	"html/template"

	"k8s.io/utils/pointer"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	meeterr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/utils"
	ctrl "sigs.k8s.io/controller-runtime"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const JicofoName = "jicofo"

type Jicofo struct {
	client.Client
	*v1beta1.Jicofo

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

func NewJicofo(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (Jitsi, error) {
	j := &v1beta1.Jicofo{}
	if err := c.Get(ctx, req.NamespacedName, j); err != nil {
		return nil, err
	}
	defaultLabels := utils.GetDefaultLabels(JicofoName)
	if !j.DeletionTimestamp.IsZero() {
		return &Jicofo{
			Client:    c,
			Jicofo:    j,
			name:      JicofoName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    defaultLabels,
		}, meeterr.UnderDeletion()
	}
	if err := addFinalizerToJicofo(ctx, c, j); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &Jicofo{
		Client:    c,
		Jicofo:    j,
		ctx:       ctx,
		log:       l,
		name:      JicofoName,
		namespace: j.Namespace,
		labels:    defaultLabels,
	}, nil
}

func addFinalizerToJicofo(ctx context.Context, c client.Client, j *v1beta1.Jicofo) error {
	if utils.ContainsString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		return nil
	}
	j.ObjectMeta.Finalizers = append(j.ObjectMeta.Finalizers, utils.MeetingFinalizer)
	return c.Update(ctx, j)
}

func (j *Jicofo) Create() error {
	if err := j.createCustomLoggingCM(); err != nil {
		j.log.Info("can't create jicofo logging config map", "error", err)
	}
	preparedDeployment := j.prepareDeployment()
	return j.Client.Create(j.ctx, preparedDeployment)
}

func (j *Jicofo) createCustomLoggingCM() error {
	logging := j.prepareLoggingCM()
	err := j.Client.Create(j.ctx, logging)
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (j *Jicofo) prepareLoggingCM() *v1.ConfigMap {
	tpl, err := template.New("log").Parse(jicofoCustomLogging)
	if err != nil {
		j.log.Info("can't template logging config", "error", err)
		return nil
	}
	var level = loggingLevelInfo
	for k := range j.Spec.Environments {
		if j.Spec.Environments[k].Name != loggingLevel {
			continue
		}
		level = j.Spec.Environments[k].Value
	}
	var b bytes.Buffer
	if executeErr := tpl.Execute(&b, level); executeErr != nil {
		j.log.Info("can't template logging config", "error", err)
		return nil
	}
	return &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "jicofo-custom-logging", Namespace: j.namespace,
		Labels: map[string]string{"app": JicofoName}},
		Data: map[string]string{"custom-logging.properties": b.String()}}
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
		Replicas: &j.Spec.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: j.labels,
			},
			Spec: v1.PodSpec{
				TerminationGracePeriodSeconds: &j.Spec.TerminationGracePeriodSeconds,
				ImagePullSecrets:              j.Spec.ImagePullSecrets,
				Volumes:                       volumes,
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
	loggingConfig := v1.Volume{Name: "custom-logging", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
		Items:                []v1.KeyToPath{{Key: "custom-logging.properties", Path: "logging.properties"}},
		LocalObjectReference: v1.LocalObjectReference{Name: "jicofo-custom-logging"}}}}
	if j.Spec.Exporter.Type == telegrafExporter {
		telegrafCM := v1.Volume{Name: "telegraf", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
			LocalObjectReference: v1.LocalObjectReference{Name: j.Spec.Exporter.ConfigMapName}}}}
		return append(volume, telegrafCM, loggingConfig)
	}
	return append(volume, loggingConfig)
}

func (j *Jicofo) prepareJicofoContainer() v1.Container {
	return v1.Container{
		Name:            JicofoName,
		Image:           j.Spec.Image,
		ImagePullPolicy: j.Spec.ImagePullPolicy,
		Env:             j.Spec.Environments,
		Resources:       j.Spec.Resources,
		SecurityContext: &j.Spec.SecurityContext,
		VolumeMounts: []v1.VolumeMount{
			{Name: "custom-logging", MountPath: "/defaults/logging.properties", SubPath: "logging.properties"}},
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
	switch j.Spec.Exporter.Type {
	case telegrafExporter:
		return v1.Container{
			Name:            exporterContainerName,
			Image:           j.Spec.Exporter.Image,
			Env:             j.Spec.Exporter.Environments,
			VolumeMounts:    []v1.VolumeMount{{Name: "telegraf", MountPath: "/etc/telegraf/"}},
			Resources:       j.Spec.Exporter.Resources,
			ImagePullPolicy: j.Spec.ImagePullPolicy,
			SecurityContext: &j.Spec.Exporter.SecurityContext,
		}
	default:
		return v1.Container{
			Name:            exporterContainerName,
			Image:           j.Spec.Exporter.Image,
			Args:            []string{"-videobridge-url", "http://localhost:8080/colibri/stats"},
			Ports:           []v1.ContainerPort{{Name: "http", ContainerPort: j.Spec.Exporter.Port, Protocol: v1.ProtocolTCP}},
			Env:             j.Spec.Exporter.Environments,
			Resources:       j.Spec.Exporter.Resources,
			ImagePullPolicy: j.Spec.ImagePullPolicy,
			SecurityContext: &v1.SecurityContext{
				RunAsUser:                pointer.Int64Ptr(defaultExporterUser),
				Privileged:               pointer.BoolPtr(false),
				RunAsNonRoot:             pointer.BoolPtr(true),
				ReadOnlyRootFilesystem:   pointer.BoolPtr(true),
				AllowPrivilegeEscalation: pointer.BoolPtr(false),
			},
		}
	}
}

func (j *Jicofo) Update() error {
	if err := j.updateCustomLoggingCM(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := j.createCustomLoggingCM(); createErr != nil {
				j.log.Info("can't create jicofo logging cm", "error", createErr)
			}
		} else {
			j.log.Info("can't update jicofo logging cm", "error", err)
		}
	}
	updatedDeployment := j.prepareDeployment()
	return j.Client.Update(j.ctx, updatedDeployment)
}

func (j *Jicofo) updateCustomLoggingCM() error {
	logging := j.prepareLoggingCM()
	return j.Client.Update(j.ctx, logging)
}

func (j *Jicofo) UpdateStatus() error { return nil }

func (j *Jicofo) Delete() error {
	if err := j.deleteCMs(); client.IgnoreNotFound(err) != nil {
		j.log.Info("failed to delete jicofo logging cm", "error", err, "namespace", j.namespace)
	}
	deployment, err := j.Get()
	if err != nil {
		return err
	}
	return j.Client.Delete(j.ctx, deployment)
}

func (j *Jicofo) deleteCMs() error {
	var cms v1.ConfigMapList
	filter := &client.ListOptions{
		LabelSelector: client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(map[string]string{"app": JicofoName})}}
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

func (j *Jicofo) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := j.Client.Get(j.ctx, types.NamespacedName{
		Namespace: j.namespace,
		Name:      j.name,
	}, deployment)
	return deployment, err
}
