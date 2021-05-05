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
