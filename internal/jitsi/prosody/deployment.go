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

package prosody

import (
	"bytes"
	"html/template"

	"github.com/onmetal/meeting-operator/internal/jitsi"
	"github.com/onmetal/meeting-operator/internal/jitsi/jvb"
	"github.com/onmetal/meeting-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const enabled = "true"

const appName = "prosody"

func (p *Prosody) Create() error {
	if svcErr := p.Service.Create(); svcErr != nil {
		return svcErr
	}
	if err := p.createTurnCM(); err != nil {
		p.log.Info("can't create prosody turn config map", "error", err)
	}
	newDeployment := p.prepareDeployment()
	return p.Client.Create(p.ctx, newDeployment)
}

func (p *Prosody) createTurnCM() error {
	turn := p.prepareTurnCredentialsCM()
	err := p.Client.Create(p.ctx, turn)
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (p *Prosody) prepareTurnCredentialsCM() *corev1.ConfigMap {
	tpl, err := template.New("log").Parse(prosodyTurnConfig)
	if err != nil {
		p.log.Info("can't template turn config", "error", err)
		return nil
	}
	config := p.getTurnCredentialsConfig()
	var b bytes.Buffer
	if executeErr := tpl.Execute(&b, config); executeErr != nil {
		p.log.Info("can't template logging config", "error", err)
		return nil
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prosody-turn-config", Namespace: p.namespace,
			Labels: map[string]string{"app": appName},
		},
		Data: map[string]string{"turn.cfg.lua": b.String()},
	}
}

func (p *Prosody) getTurnCredentialsConfig() jvb.TurnConfig {
	var config jvb.TurnConfig
	for env := range p.Spec.Environments {
		switch p.Spec.Environments[env].Name {
		case "XMPP_DOMAIN":
			config.XMPPDomain = p.Spec.Environments[env].Value
		case "TURNCREDENTIALS_SECRET":
			if p.Spec.Environments[env].ValueFrom != nil {
				config.TurnCredentials = p.getTurnCredential(p.Spec.Environments[env])
				continue
			}
			config.TurnCredentials = p.Spec.Environments[env].Value
		case "TURN_HOST":
			config.TurnHost = p.Spec.Environments[env].Value
		case "STUN_HOST":
			config.StunHost = p.Spec.Environments[env].Value
		case "TURN_PORT":
			config.TurnPort = p.Spec.Environments[env].Value
		case "STUN_PORT":
			config.StunPort = p.Spec.Environments[env].Value
		case "TURNS_PORT":
			config.TurnsPort = p.Spec.Environments[env].Value
		case "STUN_ENABLED":
			if p.Spec.Environments[env].Value == enabled {
				config.StunEnabled = true
				continue
			}
			config.StunEnabled = false
		case "TURN_UDP_ENABLED":
			if p.Spec.Environments[env].Value == enabled {
				config.TurnUDPEnabled = true
				continue
			}
			config.TurnUDPEnabled = false
		}
	}
	return config
}

func (p *Prosody) getTurnCredential(env corev1.EnvVar) string {
	switch {
	case env.ValueFrom.SecretKeyRef != nil:
		sec := &corev1.Secret{}
		if err := p.Client.Get(p.ctx, types.NamespacedName{
			Namespace: p.namespace,
			Name:      env.ValueFrom.SecretKeyRef.Name,
		}, sec); err != nil {
			p.log.Info("can't get turn secret", "error", err)
			return ""
		}
		cred, ok := sec.Data[env.ValueFrom.SecretKeyRef.Key]
		if !ok {
			p.log.Info("turn key not found in secret")
			return ""
		}
		return string(cred)
	case env.ValueFrom.ConfigMapKeyRef != nil:
		cm := &corev1.ConfigMap{}
		if err := p.Client.Get(p.ctx, types.NamespacedName{
			Namespace: p.namespace,
			Name:      env.ValueFrom.SecretKeyRef.Name,
		}, cm); err != nil {
			p.log.Info("can't get turn secret", "error", err)
			return ""
		}
		return cm.Data[env.ValueFrom.SecretKeyRef.Key]
	default:
		return ""
	}
}

func (p *Prosody) prepareDeployment() *appsv1.Deployment {
	spec := p.prepareDeploymentSpec()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.name,
			Namespace:   p.namespace,
			Labels:      p.labels,
			Annotations: p.Annotations,
		},
		Spec: spec,
	}
}

func (p *Prosody) prepareDeploymentSpec() appsv1.DeploymentSpec {
	volumes := p.prepareVolumesForProsody()
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: p.labels,
		},
		Replicas: &p.Spec.Replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: p.labels,
			},
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: &p.Spec.TerminationGracePeriodSeconds,
				ImagePullSecrets:              p.Spec.ImagePullSecrets,
				Volumes:                       volumes,
				Containers: []corev1.Container{
					{
						Name:            appName,
						Image:           p.Spec.Image,
						ImagePullPolicy: p.Spec.ImagePullPolicy,
						Env:             p.Spec.Environments,
						Ports:           jitsi.GetContainerPorts(p.Spec.Ports),
						Resources:       p.Spec.Resources,
						SecurityContext: &p.Spec.SecurityContext,
						VolumeMounts: []corev1.VolumeMount{
							{Name: "turn", MountPath: "/defaults/conf.d/turn.cfg.lua", SubPath: "turn.cfg.lua"},
						},
					},
				},
			},
		},
	}
}

func (p *Prosody) prepareVolumesForProsody() []corev1.Volume {
	var volume []corev1.Volume
	loggingConfig := corev1.Volume{Name: "turn", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
		Items:                []corev1.KeyToPath{{Key: "turn.cfg.lua", Path: "turn.cfg.lua"}},
		LocalObjectReference: corev1.LocalObjectReference{Name: "prosody-turn-config"},
	}}}
	return append(volume, loggingConfig)
}

func (p *Prosody) Update(deployment *appsv1.Deployment) error {
	if err := p.Service.Update(); err != nil {
		return err
	}
	if err := p.updateTurnCM(); err != nil {
		if apierrors.IsNotFound(err) {
			if createErr := p.createTurnCM(); createErr != nil {
				p.log.Info("can't create prosody turn cm", "error", createErr)
			}
		} else {
			p.log.Info("can't update prosody turn cm", "error", err)
		}
	}
	deployment.Annotations = p.Annotations
	deployment.Labels = p.Labels
	deployment.Spec = p.prepareDeploymentSpec()
	return p.Client.Update(p.ctx, deployment)
}

func (p *Prosody) updateTurnCM() error {
	logging := p.prepareTurnCredentialsCM()
	return p.Client.Update(p.ctx, logging)
}

func (p *Prosody) Delete() error {
	if err := utils.RemoveFinalizer(p.ctx, p.Client, p.Prosody); err != nil {
		p.log.Info("can't remove finalizer", "error", err)
	}
	if err := p.Service.Delete(); err != nil {
		return err
	}
	if err := p.deleteCMs(); client.IgnoreNotFound(err) != nil {
		p.log.Info("failed to delete prosody logging cm", "error", err, "namespace", p.namespace)
	}
	deployment, err := p.Get()
	if err != nil {
		return err
	}
	return p.Client.Delete(p.ctx, deployment)
}

func (p *Prosody) deleteCMs() error {
	var cms corev1.ConfigMapList
	filter := &client.ListOptions{
		LabelSelector: client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(map[string]string{"app": appName})},
	}
	if err := p.Client.List(p.ctx, &cms, filter); err != nil {
		return err
	}
	for cm := range cms.Items {
		if err := p.Client.Delete(p.ctx, &cms.Items[cm]); err != nil {
			p.log.Info("can't delete config map", "appName", cms.Items[cm].Name, "error", err)
		}
	}
	return nil
}

func (p *Prosody) Get() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := p.Client.Get(p.ctx, types.NamespacedName{
		Namespace: p.namespace,
		Name:      p.name,
	}, deployment)
	return deployment, err
}
