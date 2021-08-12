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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const enabled = "true"

const ProsodyName = "prosody"

type Prosody struct {
	client.Client
	*v1beta1.Prosody
	*service

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

func NewProsody(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (Jitsi, error) {
	p := &v1beta1.Prosody{}
	if err := c.Get(ctx, req.NamespacedName, p); err != nil {
		return nil, err
	}
	defaultLabels := utils.GetDefaultLabels(ProsodyName)
	s := newService(ctx, c, l, ProsodyName, p.Namespace, p.Spec.ServiceAnnotations, defaultLabels, p.Spec.ServiceType, p.Spec.Ports)
	if !p.DeletionTimestamp.IsZero() {
		return &Prosody{
			Client:    c,
			Prosody:   p,
			service:   s,
			name:      ProsodyName,
			namespace: p.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    defaultLabels,
		}, meeterr.UnderDeletion()
	}
	if err := addFinalizerToProsody(ctx, c, p); err != nil {
		l.Info("finalizer cannot be added", "error", err)
	}
	return &Prosody{
		Client:    c,
		Prosody:   p,
		service:   s,
		name:      ProsodyName,
		namespace: p.Namespace,
		ctx:       ctx,
		log:       l,
		labels:    defaultLabels,
	}, nil
}

func addFinalizerToProsody(ctx context.Context, c client.Client, j *v1beta1.Prosody) error {
	if utils.ContainsString(j.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		return nil
	}
	j.ObjectMeta.Finalizers = append(j.ObjectMeta.Finalizers, utils.MeetingFinalizer)
	return c.Update(ctx, j)
}

func (p *Prosody) Create() error {
	if err := p.service.Create(); err != nil {
		return err
	}
	if err := p.createTurnCM(); err != nil {
		p.log.Info("can't create prosody turn config map", "error", err)
	}
	preparedDeployment := p.prepareDeployment()
	return p.Client.Create(p.ctx, preparedDeployment)
}

func (p *Prosody) createTurnCM() error {
	turn := p.prepareTurnCredentialsCM()
	err := p.Client.Create(p.ctx, turn)
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (p *Prosody) prepareTurnCredentialsCM() *v1.ConfigMap {
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
	return &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "prosody-turn-config", Namespace: p.namespace,
		Labels: map[string]string{"app": ProsodyName}},
		Data: map[string]string{"turn.cfg.lua": b.String()}}
}

func (p *Prosody) getTurnCredentialsConfig() TurnConfig {
	var config TurnConfig
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

func (p *Prosody) getTurnCredential(env v1.EnvVar) string {
	switch {
	case env.ValueFrom.SecretKeyRef != nil:
		sec := &v1.Secret{}
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
		cm := &v1.ConfigMap{}
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
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: p.labels,
			},
			Spec: v1.PodSpec{
				TerminationGracePeriodSeconds: &p.Spec.TerminationGracePeriodSeconds,
				ImagePullSecrets:              p.Spec.ImagePullSecrets,
				Volumes:                       volumes,
				Containers: []v1.Container{
					{
						Name:            ProsodyName,
						Image:           p.Spec.Image,
						ImagePullPolicy: p.Spec.ImagePullPolicy,
						Env:             p.Spec.Environments,
						Ports:           getContainerPorts(p.Spec.Ports),
						Resources:       p.Spec.Resources,
						SecurityContext: &p.Spec.SecurityContext,
						VolumeMounts: []v1.VolumeMount{
							{Name: "turn", MountPath: "/defaults/conf.d/turn.cfg.lua", SubPath: "turn.cfg.lua"}},
					},
				},
			},
		},
	}
}

func (p *Prosody) prepareVolumesForProsody() []v1.Volume {
	var volume []v1.Volume
	loggingConfig := v1.Volume{Name: "turn", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
		Items:                []v1.KeyToPath{{Key: "turn.cfg.lua", Path: "turn.cfg.lua"}},
		LocalObjectReference: v1.LocalObjectReference{Name: "prosody-turn-config"}}}}
	return append(volume, loggingConfig)
}

func (p *Prosody) Update() error {
	if err := p.service.Update(); err != nil {
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
	updatedDeployment := p.prepareDeployment()
	return p.Client.Update(p.ctx, updatedDeployment)
}

func (p *Prosody) updateTurnCM() error {
	logging := p.prepareTurnCredentialsCM()
	return p.Client.Update(p.ctx, logging)
}

func (p *Prosody) UpdateStatus() error { return nil }

func (p *Prosody) Delete() error {
	if utils.ContainsString(p.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		p.ObjectMeta.Finalizers = utils.RemoveString(p.ObjectMeta.Finalizers, utils.MeetingFinalizer)
		if err := p.Client.Update(p.ctx, p.Prosody); err != nil {
			p.log.Info("can't update prosody cr", "error", err)
		}
	}
	if err := p.service.Delete(); err != nil {
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
	var cms v1.ConfigMapList
	filter := &client.ListOptions{
		LabelSelector: client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(map[string]string{"app": ProsodyName})}}
	if err := p.Client.List(p.ctx, &cms, filter); err != nil {
		return err
	}
	for cm := range cms.Items {
		if err := p.Client.Delete(p.ctx, &cms.Items[cm]); err != nil {
			p.log.Info("can't delete config map", "name", cms.Items[cm].Name, "error", err)
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
