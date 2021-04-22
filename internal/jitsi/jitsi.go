package jitsi

import (
	"context"
	"github.com/onmetal/meeting-operator/internal/utils"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	JicofoName  = "jicofo"
	ProsodyName = "prosody"
	WebName     = "web"
)

type Jitsi interface {
	Create() error
	Update() error
	Delete() error
}

type Web struct {
	client.Client
	*v1alpha1.Web

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

type Prosody struct {
	client.Client
	*v1alpha1.Prosody

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

type Jicofo struct {
	client.Client
	*v1alpha1.Jicofo

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
}

type JVB struct {
	client.Client
	*v1alpha1.JVB

	ctx                             context.Context
	log                             logr.Logger
	podName, serviceName, namespace string
	replica                         int32
}

func NewJitsi(ctx context.Context, appName string,
	j *v1alpha1.Jitsi, c client.Client, l logr.Logger) Jitsi {
	switch appName {
	case WebName:
		labels := utils.GetDefaultLabels(WebName)
		return &Web{
			Client:    c,
			Web:       &j.Spec.Web,
			name:      WebName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    labels,
		}
	case ProsodyName:
		labels := utils.GetDefaultLabels(ProsodyName)
		return &Prosody{
			Client:    c,
			Prosody:   &j.Spec.Prosody,
			name:      ProsodyName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    labels,
		}
	case JicofoName:
		labels := utils.GetDefaultLabels(JicofoName)
		return &Jicofo{
			Client:    c,
			Jicofo:    &j.Spec.Jicofo,
			name:      JicofoName,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
			labels:    labels,
		}
	case JvbContainerName:
		return &JVB{
			Client:    c,
			JVB:       &j.Spec.JVB,
			namespace: j.Namespace,
			ctx:       ctx,
			log:       l,
		}
	default:
		return nil
	}
}
