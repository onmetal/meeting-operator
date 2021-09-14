package jitsiautoscaler

import (
	"context"
	"math"
	"time"

	"github.com/onmetal/meeting-operator/apis/jitsiautoscaler/v1alpha1"
	"github.com/prometheus/common/model"
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

func (p *prom) Scale() {
	ctx, cancel := context.WithTimeout(p.ctx, promRequestTimeout)
	defer cancel()
	p.ctx = ctx
	target := float64(p.Spec.Metric.TargetAverageUtilization)
	avg := p.getAvgValueForMetric(p.Spec.Metric.Name)
	if target > avg {
		desiredReplicas := int32(math.RoundToEven(avg / target))
		if err := p.scaleDown(desiredReplicas); err != nil {
			p.log.Info("can't scale down", "error", err)
		}
		return
	}
	desiredReplicas := int32(math.RoundToEven(avg / target))
	if err := p.scaleUp(desiredReplicas); err != nil {
		p.log.Info("can't scale up", "error", err)
	}
}

func (p *prom) getAvgValueForMetric(name v1alpha1.MetricName) float64 {
	switch name {
	case v1alpha1.ResourceCPU:
		return p.countAvgValueByRequest(promCPURequest)
	case v1alpha1.ResourceConference:
		return p.countAvgValueByRequest(promConferenceRequest)
	case v1alpha1.ResourceParticipants:
		return p.countAvgValueByRequest(promParticipantRequest)
	default:
		return 0
	}
}

func (p *prom) countAvgValueByRequest(request string) float64 {
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

func (p *prom) scaleUp(desiredReplicas int32) error {
	jitsi, getErr := getJVBCR(p.ctx, p.Client, p.Spec.ScaleTargetRef.Name, p.Namespace)
	if getErr != nil {
		return getErr
	}
	jitsi.Spec.Replicas *= desiredReplicas
	if jitsi.Spec.Replicas > p.Spec.MaxReplicas {
		jitsi.Spec.Replicas = p.Spec.MaxReplicas
	}
	return p.Update(p.ctx, jitsi)
}

func (p *prom) scaleDown(desiredReplicas int32) error {
	jitsi, getErr := getJVBCR(p.ctx, p.Client, p.Spec.ScaleTargetRef.Name, p.Namespace)
	if getErr != nil {
		return getErr
	}
	jitsi.Spec.Replicas = desiredReplicas
	if jitsi.Spec.Replicas < p.Spec.MinReplicas {
		jitsi.Spec.Replicas = p.Spec.MinReplicas
	}
	return p.Update(p.ctx, jitsi)
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
