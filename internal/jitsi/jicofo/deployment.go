// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package jicofo

import (
	"bytes"
	"html/template"

	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const appName = "jicofo"

const (
	telegrafExporter            = "telegraf"
	exporterContainerName       = "exporter"
	defaultExporterUser   int64 = 10001
	healthPort                  = 8888
)

const (
	initialDelaySeconds = 30
	timeoutSeconds      = 30
	periodSeconds       = 15
	successThreshold    = 1
	failureThreshold    = 3
)

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

func (j *Jicofo) prepareLoggingCM() *corev1.ConfigMap {
	tpl, err := template.New("log").Parse(jicofoCustomLogging)
	if err != nil {
		j.log.Info("can't template logging config", "error", err)
		return nil
	}
	level := loggingLevelInfo
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
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "jicofo-custom-logging", Namespace: j.namespace,
			Labels: map[string]string{"app": appName},
		},
		Data: map[string]string{"custom-logging.properties": b.String()},
	}
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
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: j.labels,
			},
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: &j.Spec.TerminationGracePeriodSeconds,
				ImagePullSecrets:              j.Spec.ImagePullSecrets,
				Volumes:                       volumes,
				Containers: []corev1.Container{
					jicofo,
					exporter,
				},
			},
		},
	}
}

func (j *Jicofo) prepareVolumesForJicofo() []corev1.Volume {
	var volume []corev1.Volume
	loggingConfig := corev1.Volume{Name: "custom-logging", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
		Items:                []corev1.KeyToPath{{Key: "custom-logging.properties", Path: "logging.properties"}},
		LocalObjectReference: corev1.LocalObjectReference{Name: "jicofo-custom-logging"},
	}}}
	if j.Spec.Exporter.Type == telegrafExporter {
		telegrafCM := corev1.Volume{Name: "telegraf", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: j.Spec.Exporter.ConfigMapName},
		}}}
		return append(volume, telegrafCM, loggingConfig)
	}
	return append(volume, loggingConfig)
}

func (j *Jicofo) prepareJicofoContainer() corev1.Container {
	return corev1.Container{
		Name:            appName,
		Image:           j.Spec.Image,
		ImagePullPolicy: j.Spec.ImagePullPolicy,
		Env:             j.Spec.Environments,
		Resources:       j.Spec.Resources,
		SecurityContext: &j.Spec.SecurityContext,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "custom-logging", MountPath: "/defaults/logging.properties", SubPath: "logging.properties"},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/about/health",
					Port:   intstr.IntOrString{IntVal: healthPort},
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: initialDelaySeconds,
			TimeoutSeconds:      timeoutSeconds,
			PeriodSeconds:       periodSeconds,
			SuccessThreshold:    successThreshold,
			FailureThreshold:    failureThreshold,
		},
	}
}

func (j *Jicofo) prepareExporterContainer() corev1.Container {
	switch j.Spec.Exporter.Type {
	case telegrafExporter:
		return corev1.Container{
			Name:            exporterContainerName,
			Image:           j.Spec.Exporter.Image,
			Env:             j.Spec.Exporter.Environments,
			VolumeMounts:    []corev1.VolumeMount{{Name: "telegraf", MountPath: "/etc/telegraf/"}},
			Resources:       j.Spec.Exporter.Resources,
			ImagePullPolicy: j.Spec.ImagePullPolicy,
			SecurityContext: &j.Spec.Exporter.SecurityContext,
		}
	default:
		return corev1.Container{
			Name:            exporterContainerName,
			Image:           j.Spec.Exporter.Image,
			Args:            []string{"-videobridge-url", "http://localhost:8080/colibri/stats"},
			Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: j.Spec.Exporter.Port, Protocol: corev1.ProtocolTCP}},
			Env:             j.Spec.Exporter.Environments,
			Resources:       j.Spec.Exporter.Resources,
			ImagePullPolicy: j.Spec.ImagePullPolicy,
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:                ptr.To(defaultExporterUser),
				Privileged:               ptr.To(false),
				RunAsNonRoot:             ptr.To(true),
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
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
	if err := utils.RemoveFinalizer(j.ctx, j.Client, j.Jicofo); err != nil {
		j.log.Info("can't remove finalizer", "error", err)
	}
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
	var cms corev1.ConfigMapList
	filter := &client.ListOptions{
		LabelSelector: client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(map[string]string{"app": appName})},
	}
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
