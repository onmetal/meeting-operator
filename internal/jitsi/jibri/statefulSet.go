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

package jibri

import (
	"fmt"

	"github.com/onmetal/meeting-operator/internal/jitsi"
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const appName = "jibri"

func (j *Jibri) Create() error {
	preparedSTS := j.prepareSTS()
	return j.Client.Create(j.ctx, preparedSTS)
}

func (j *Jibri) prepareSTS() *appsv1.StatefulSet {
	spec := j.prepareSTSSpec()
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        j.name,
			Namespace:   j.namespace,
			Labels:      j.labels,
			Annotations: j.Annotations,
		},
		Spec: spec,
	}
}

func (j *Jibri) prepareSTSSpec() appsv1.StatefulSetSpec {
	sts := appsv1.StatefulSetSpec{
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
				Containers: []v1.Container{
					{
						Name:            appName,
						Image:           j.Spec.Image,
						ImagePullPolicy: j.Spec.ImagePullPolicy,
						Env:             j.Spec.Environments,
						Ports:           jitsi.GetContainerPorts(j.Spec.Ports),
						Resources:       j.Spec.Resources,
						SecurityContext: &j.Spec.SecurityContext,
					},
				},
			},
		},
	}
	j.setPV(&sts)
	return sts
}

func (j *Jibri) setPV(sts *appsv1.StatefulSetSpec) {
	switch {
	case j.Spec.Storage == nil:
		sts.VolumeClaimTemplates = nil
		sts.Template.Spec.Volumes = append(sts.Template.Spec.Volumes, v1.Volume{
			Name: "snd",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/dev/snd",
				},
			},
		})
		sts.Template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
			{
				Name:      "snd",
				MountPath: "/dev/snd",
			},
		}
	case j.Spec.Storage.PVC.Spec.Resources.Requests != nil:
		pvc := j.preparePVC()
		sts.VolumeClaimTemplates = []v1.PersistentVolumeClaim{pvc}
		sts.Template.Spec.Volumes = append(sts.Template.Spec.Volumes, v1.Volume{
			Name: "snd",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/dev/snd",
				},
			},
		})
		sts.Template.Spec.Containers[0].VolumeMounts = j.setVolumeMounts()
	case j.Spec.Storage.EmptyDir.SizeLimit != nil && j.Spec.Storage != nil:
		sts.Template.Spec.Volumes = append(sts.Template.Spec.Volumes,
			v1.Volume{
				Name: volumeName(j.name),
				VolumeSource: v1.VolumeSource{
					EmptyDir: j.Spec.Storage.EmptyDir,
				},
			},
			v1.Volume{
				Name: "snd",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/dev/snd",
					},
				},
			})
		sts.Template.Spec.Containers[0].VolumeMounts = j.setVolumeMounts()
	default:
		sts.VolumeClaimTemplates = nil
		sts.Template.Spec.Volumes = append(sts.Template.Spec.Volumes, v1.Volume{
			Name: "snd",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/dev/snd",
				},
			},
		})
		sts.Template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
			{
				Name:      "snd",
				MountPath: "/dev/snd",
			},
		}
	}
}

func (j *Jibri) preparePVC() v1.PersistentVolumeClaim {
	if j.Spec.Storage.PVC.Kind == "" {
		j.Spec.Storage.PVC.Kind = "PersistentVolumeClaim"
	}
	if j.Spec.Storage.PVC.APIVersion == "" {
		j.Spec.Storage.PVC.APIVersion = "v1"
	}
	if j.Spec.Storage.PVC.Name == "" {
		j.Spec.Storage.PVC.Name = volumeName(j.name)
	}
	if j.Spec.Storage.PVC.Spec.AccessModes == nil {
		j.Spec.Storage.PVC.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	}
	return v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       j.Spec.Storage.PVC.Kind,
			APIVersion: j.Spec.Storage.PVC.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.Spec.Storage.PVC.Name,
			Namespace: j.namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes:      j.Spec.Storage.PVC.Spec.AccessModes,
			Resources:        j.Spec.Storage.PVC.Spec.Resources,
			VolumeName:       j.Spec.Storage.PVC.Name,
			StorageClassName: j.Spec.Storage.PVC.Spec.StorageClassName,
		},
	}
}

func (j *Jibri) setVolumeMounts() []v1.VolumeMount {
	mountPath := "/config/recordings/"
	for env := range j.Spec.Environments {
		if j.Spec.Environments[env].Name != "JIBRI_RECORDING_DIR" {
			continue
		}
		mountPath = j.Spec.Environments[env].Value
	}
	return []v1.VolumeMount{
		{
			Name:      volumeName(j.name),
			ReadOnly:  false,
			MountPath: mountPath,
		},
		{
			Name:      "snd",
			MountPath: "/dev/snd",
		},
	}
}

func (j *Jibri) Update(sts *appsv1.StatefulSet) error {
	sts.Annotations = j.Annotations
	sts.Labels = j.labels
	sts.Spec = j.prepareSTSSpec()
	return j.Client.Update(j.ctx, sts)
}

func (j *Jibri) Delete() error {
	if err := utils.RemoveFinalizer(j.ctx, j.Client, j.Jibri); err != nil {
		j.log.Info("can't remove finalizer", "error", err)
	}
	sts, err := j.Get()
	if err != nil {
		return err
	}
	return j.Client.Delete(j.ctx, sts)
}

func (j *Jibri) Get() (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{}
	err := j.Client.Get(j.ctx, types.NamespacedName{
		Namespace: j.namespace,
		Name:      j.name,
	}, sts)
	return sts, err
}

func volumeName(name string) string {
	return fmt.Sprintf("%s-db", name)
}
