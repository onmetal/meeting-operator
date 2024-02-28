// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package jitsiautoscaler

import (
	"fmt"
	"math"
	"time"

	influxdb2api "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/onmetal/meeting-operator/apis/jitsiautoscaler/v1alpha1"
)

const (
	influxQuery               = `from(bucket: "%s")|> range(start: -15m) |>filter(fn: (r) => r["_measurement"] == "jitsi_stats")|> filter(fn: (r) => r["_field"] == "%s")|> distinct(column: "_value")` //nolint:lll //reason: would be removed
	influxCPUMetrics          = "cpu"
	influxConferencesMetrics  = "conferences"
	influxParticipantsMetrics = "participants"
)

func (i *influx) Scale() {
	target := float64(i.Spec.Metric.TargetAverageUtilization)
	avg := i.getAvgValueForMetric(i.Spec.Metric.Name)
	if target > avg {
		desiredReplicas := int32(math.RoundToEven(avg / target))
		if err := i.scaleDown(desiredReplicas); err != nil {
			i.log.Info("can't scale down", "error", err)
		}
		return
	}
	desiredReplicas := int32(math.RoundToEven(avg / target))
	if err := i.scaleUp(desiredReplicas); err != nil {
		i.log.Info("can't scale up", "error", err)
	}
}

func (i *influx) getAvgValueForMetric(name v1alpha1.MetricName) float64 {
	switch name {
	case v1alpha1.ResourceCPU:
		return i.countAvgValueByRequest(influxCPUMetrics)
	case v1alpha1.ResourceConference:
		return i.countAvgValueByRequest(influxConferencesMetrics)
	case v1alpha1.ResourceParticipants:
		return i.countAvgValueByRequest(influxParticipantsMetrics)
	default:
		return 0
	}
}

func (i *influx) countAvgValueByRequest(field string) float64 {
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

func (i *influx) scaleUp(desiredReplicas int32) error {
	jitsi, getErr := getJVBCR(i.ctx, i.Client, i.Spec.ScaleTargetRef.Name, i.Namespace)
	if getErr != nil {
		return getErr
	}
	jitsi.Spec.Replicas *= desiredReplicas
	if jitsi.Spec.Replicas > i.Spec.MaxReplicas {
		jitsi.Spec.Replicas = i.Spec.MaxReplicas
	}
	return i.Update(i.ctx, jitsi)
}

func (i *influx) scaleDown(desiredReplicas int32) error {
	jitsi, getErr := getJVBCR(i.ctx, i.Client, i.Spec.ScaleTargetRef.Name, i.Namespace)
	if getErr != nil {
		return getErr
	}
	jitsi.Spec.Replicas = desiredReplicas
	if jitsi.Spec.Replicas < i.Spec.MinReplicas {
		jitsi.Spec.Replicas = i.Spec.MinReplicas
	}
	return i.Update(i.ctx, jitsi)
}

func (i *influx) Repeat() time.Duration {
	if i.Spec.Interval == "" {
		return defaultRepeatInterval
	}
	interval, err := time.ParseDuration(i.Spec.Interval)
	if err != nil {
		i.log.Info("can't parse duration", "error", err)
		return defaultRepeatInterval
	}
	return interval
}
