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
	"fmt"
	"math"
	"time"

	influxdb2api "github.com/influxdata/influxdb-client-go/v2/api"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	jitsiv1alpha "github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"
	"k8s.io/apimachinery/pkg/types"

	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/prometheus/common/model"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/jitsiautoscaler/v1alpha1"
	promapi "github.com/prometheus/client_golang/api"
	promv1api "github.com/prometheus/client_golang/api/prometheus/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	promRangeStepMinute  = 15 * time.Minute
	promRangeStartMinute = 15 * time.Minute
	promRequestTimeout   = 30 * time.Second
)

const (
	promCPURequest         = `rate(container_cpu_usage_seconds_total{container="jvb", id=~"/kubelet.*"}[5m])`
	promConferenceRequest  = `jitsi_conferences{job=~"exporter-jvb-.*"}`
	promParticipantRequest = `jitsi_participants{job=~"exporter-jvb-.*"}`
)

const (
	influxQuery               = `from(bucket: "%s")|> range(start: -15m) |>filter(fn: (r) => r["_measurement"] == "jitsi_stats")|> filter(fn: (r) => r["_field"] == "%s")|> distinct(column: "_value")` //nolint:lll
	influxCPUMetrics          = "cpu"
	influxConferencesMetrics  = "conferences"
	influxParticipantsMetrics = "participants"
)

const defaultRepeatIntervalSecond = 600 * time.Second

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

func New(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (AutoScaler, error) {
	var p *prom
	var influxdb *influx
	jas := &v1alpha1.AutoScaler{}
	if err := c.Get(ctx, req.NamespacedName, jas); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, meetingerr.NotExist(req.Name)
		}
		return nil, err
	}
	switch jas.Spec.Type {
	case "prometheus":
		promClient, err := promapi.NewClient(promapi.Config{
			Address: jas.Spec.Host,
		})
		if err != nil {
			return nil, err
		}
		timeRange := promv1api.Range{
			Start: time.Now().Add(-promRangeStartMinute),
			End:   time.Now(),
			Step:  promRangeStepMinute,
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
			return nil, errors.New("token not exist")
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
		return nil, meetingerr.NotExist(jas.Spec.Type)
	}
}

func (p *prom) Scale() {
	ctx, cancel := context.WithTimeout(p.ctx, promRequestTimeout)
	defer cancel()
	p.ctx = ctx
	for m := range p.Spec.Metrics {
		target := float64(p.Spec.Metrics[m].Resource.TargetAverageUtilization)
		avg := p.metricsCounting(p.Spec.Metrics[m].Resource.Name)
		if target > avg {
			desiredReplicas := math.RoundToEven(avg / target)
			if err := scaleDown(p.ctx, p.Client, p.Spec.ScaleTargetRef.Name,
				p.Namespace, int32(desiredReplicas), p.Spec.MinReplicas); err != nil {
				p.log.Info("can't scale down", "error", err)
			}
			continue
		}
		desiredReplicas := math.RoundToEven(avg / target)
		if err := scaleUp(p.ctx, p.Client, p.Spec.ScaleTargetRef.Name,
			p.Namespace, int32(desiredReplicas), p.Spec.MaxReplicas); err != nil {
			p.log.Info("can't scale up", "error", err)
		}
	}
}

func (p *prom) metricsCounting(resource v1alpha1.ResourceName) float64 {
	switch resource {
	case v1alpha1.ResourceCPU:
		return p.countAvg(promCPURequest)
	case v1alpha1.ResourceConference:
		return p.countAvg(promConferenceRequest)
	case v1alpha1.ResourceParticipants:
		return p.countAvg(promParticipantRequest)
	default:
		return 0
	}
}

func (p *prom) countAvg(request string) float64 {
	result, _, err := p.apiv1.QueryRange(p.ctx, request, p.timeRange)
	if err != nil {
		p.log.Info("can't query prometheus", "error", err)
		return 1
	}
	var sum model.SampleValue
	for res := range result.(model.Matrix) {
		sum = +result.(model.Matrix)[res].Values[1].Value
	}
	return float64(sum / model.SampleValue(len(result.(model.Matrix))))
}

func (p *prom) Repeat() time.Duration {
	if p.Spec.Interval == "" {
		return defaultRepeatIntervalSecond
	}
	interval, err := time.ParseDuration(p.Spec.Interval)
	if err != nil {
		p.log.Info("can't parse duration", "error", err)
		return defaultRepeatIntervalSecond
	}
	return interval
}

func (i *influx) Scale() {
	for m := range i.Spec.Metrics {
		target := float64(i.Spec.Metrics[m].Resource.TargetAverageUtilization)
		avg := i.metricsCounting(i.Spec.Metrics[m].Resource.Name)
		if target > avg {
			desiredReplicas := math.RoundToEven(avg / target)
			if err := scaleDown(i.ctx, i.Client, i.Spec.ScaleTargetRef.Name,
				i.Namespace, int32(desiredReplicas), i.Spec.MinReplicas); err != nil {
				i.log.Info("can't scale down", "error", err)
			}
			continue
		}
		desiredReplicas := math.RoundToEven(avg / target)
		if err := scaleUp(i.ctx, i.Client, i.Spec.ScaleTargetRef.Name,
			i.Namespace, int32(desiredReplicas), i.Spec.MaxReplicas); err != nil {
			i.log.Info("can't scale up", "error", err)
		}
	}
}

func (i *influx) metricsCounting(resource v1alpha1.ResourceName) float64 {
	switch resource {
	case v1alpha1.ResourceCPU:
		return i.countAvg(influxCPUMetrics)
	case v1alpha1.ResourceConference:
		return i.countAvg(influxConferencesMetrics)
	case v1alpha1.ResourceParticipants:
		return i.countAvg(influxParticipantsMetrics)
	default:
		return 0
	}
}

func (i *influx) countAvg(field string) float64 {
	var sum, count, value float64
	var ok bool
	query := fmt.Sprintf(influxQuery, i.bucket, field)
	result, err := i.iclient.QueryAPI(i.org).Query(i.ctx, query)
	if err != nil {
		i.log.Info("can't query influx database", "error", err)
		return 0
	}
	defer func(result *influxdb2api.QueryTableResult) {
		if closeErr := result.Close(); closeErr != nil {
			i.log.Info("can't close connection to influx properly", "error", err)
		}
	}(result)
	for result.Next() {
		values := result.Record().Values()
		value, ok = values["_value"].(float64)
		if !ok {
			count++
			continue
		}
		sum = +value
		count++
	}
	return sum / count
}

func (i *influx) Repeat() time.Duration {
	if i.Spec.Interval == "" {
		return defaultRepeatIntervalSecond
	}
	interval, err := time.ParseDuration(i.Spec.Interval)
	if err != nil {
		i.log.Info("can't parse duration", "error", err)
		return defaultRepeatIntervalSecond
	}
	return interval
}

func scaleUp(ctx context.Context, c client.Client, name, namespace string, desiredReplicas, maxReplicas int32) error {
	jitsi, getErr := getJitsiCR(ctx, c, name, namespace)
	if getErr != nil {
		return getErr
	}
	jitsi.Spec.JVB.Replicas *= desiredReplicas
	if jitsi.Spec.JVB.Replicas > maxReplicas {
		jitsi.Spec.JVB.Replicas = maxReplicas
	}
	return c.Update(ctx, jitsi)
}

func scaleDown(ctx context.Context, c client.Client, name, namespace string, desiredReplicas, minReplicas int32) error {
	jitsi, getErr := getJitsiCR(ctx, c, name, namespace)
	if getErr != nil {
		return getErr
	}
	jitsi.Spec.JVB.Replicas = desiredReplicas
	if jitsi.Spec.JVB.Replicas < minReplicas {
		jitsi.Spec.JVB.Replicas = minReplicas
	}
	return c.Update(ctx, jitsi)
}

func getJitsiCR(ctx context.Context, c client.Client, name, namespace string) (*jitsiv1alpha.Jitsi, error) {
	jitsi := &jitsiv1alpha.Jitsi{}
	keyObj := ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}}
	if getErr := c.Get(ctx, keyObj.NamespacedName, jitsi); getErr != nil {
		return nil, getErr
	}
	return jitsi, nil
}
