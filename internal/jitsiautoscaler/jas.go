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

package jitsiautoscaler

import (
	"context"
	"errors"
	"time"

	"github.com/go-logr/logr"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	"github.com/onmetal/meeting-operator/apis/jitsiautoscaler/v1alpha1"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	promapi "github.com/prometheus/client_golang/api"
	promv1api "github.com/prometheus/client_golang/api/prometheus/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultRepeatInterval = 600 * time.Second

var errTokenNotExist = errors.New("token not exist")

type AutoScaler interface {
	Scale()
	Repeat() time.Duration
}

type prom struct {
	client.Client
	*v1alpha1.AutoScaler

	ctx       context.Context
	log       logr.Logger
	apiv1     promv1api.API
	timeRange promv1api.Range
}

type influx struct {
	client.Client
	*v1alpha1.AutoScaler

	ctx         context.Context
	log         logr.Logger
	iclient     influxdb2.Client
	org, bucket string
}

func newInstance(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (AutoScaler, error) {
	var p *prom
	var influxdb *influx
	jas := &v1alpha1.AutoScaler{}
	if err := c.Get(ctx, req.NamespacedName, jas); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, meetingerr.NotExist(req.Name)
		}
		return nil, err
	}
	switch jas.Spec.MonitoringType {
	case "prometheus":
		promClient, err := promapi.NewClient(promapi.Config{
			Address: jas.Spec.Host,
		})
		if err != nil {
			return nil, err
		}
		timeRange := promv1api.Range{
			Start: time.Now().Add(-promRangeStart),
			End:   time.Now(),
			Step:  promRangeStep,
		}
		v1api := promv1api.NewAPI(promClient)
		p = &prom{
			Client:     c,
			AutoScaler: jas,
			ctx:        ctx,
			log:        l,
			apiv1:      v1api,
			timeRange:  timeRange,
		}
		return p, nil
	case "influxdb":
		var org, bucket string
		var ok bool
		org, ok = jas.Annotations["jas.influxdb/org"]
		if !ok {
			l.Info("influx org not provided, setting default to `influxdata`")
			org = "influxdata"
		}
		bucket, ok = jas.Annotations["jas.influxdb/bucket"]
		if !ok {
			l.Info("influx bucket not provided, setting default to `jitsi`")
			bucket = "jitsi"
		}
		token, ok := jas.Annotations["jas.influxdb/token"]
		if !ok {
			return nil, errTokenNotExist
		}
		influxClient := influxdb2.NewClient(jas.Spec.Host, token)
		influxdb = &influx{
			Client:     c,
			AutoScaler: jas,
			ctx:        ctx,
			log:        l,
			iclient:    influxClient,
			org:        org,
			bucket:     bucket,
		}
		return influxdb, nil
	default:
		return nil, meetingerr.NotExist(jas.Spec.MonitoringType)
	}
}

func getJVBCR(ctx context.Context, c client.Client, name, namespace string) (*v1beta1.JVB, error) {
	jitsi := &v1beta1.JVB{}
	keyObj := ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}}
	if getErr := c.Get(ctx, keyObj.NamespacedName, jitsi); getErr != nil {
		return nil, getErr
	}
	return jitsi, nil
}
