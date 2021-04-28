package etherpad

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/etherpad/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Etherpad interface {
	Create() error
	Update() error
	Delete() error
}

type Deployment struct {
	client.Client

	ctx context.Context
	log logr.Logger
	e   *v1alpha1.Etherpad
}

type Service struct {
	client.Client

	ctx context.Context
	log logr.Logger
	e   *v1alpha1.Etherpad
}

func NewEtherpad(ctx context.Context, c client.Client, l logr.Logger, e *v1alpha1.Etherpad) Etherpad {
	return &Deployment{
		Client: c,
		ctx:    ctx,
		log:    l,
		e:      e,
	}
}

func NewService(ctx context.Context, c client.Client, l logr.Logger, e *v1alpha1.Etherpad) Etherpad {
	return &Service{
		Client: c,
		ctx:    ctx,
		log:    l,
		e:      e,
	}
}

func getDefaultLabels() map[string]string {
	var defaultLabels = make(map[string]string)
	defaultLabels["app.kubernetes.io/appName"] = applicationName
	return defaultLabels
}
