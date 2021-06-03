package jitsiautoscaler

import (
	"context"
	"math"
	"time"

	jitsiv1alpha "github.com/onmetal/meeting-operator/apis/jitsi/v1alpha1"
	"k8s.io/apimachinery/pkg/types"

	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/utils"
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

type AutoScaler interface {
	Watch() error
	Stop() error
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

	ctx context.Context
	log logr.Logger
}

func New(ctx context.Context, c client.Client, l logr.Logger, req ctrl.Request) (AutoScaler, error) {
	var p *prom
	var influxdb *influx
	jas := &v1alpha1.AutoScaler{}
	if err := c.Get(ctx, req.NamespacedName, jas); err != nil {
		return nil, err
	}
	if err := addFinalizer(ctx, c, jas); err != nil {
		l.Info("can't add finalizer to jitsi autoscaler", "error", err)
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
		if !jas.DeletionTimestamp.IsZero() {
			return p, meetingerr.UnderDeletion()
		}
		return p, nil
	case "influxdb":
		influxdb = &influx{
			Client:     c,
			AutoScaler: jas,
			ctx:        ctx,
			log:        l,
		}
		if !jas.DeletionTimestamp.IsZero() {
			return influxdb, meetingerr.UnderDeletion()
		}
		return influxdb, nil
	default:
		return nil, meetingerr.NotExist(jas.Spec.Type)
	}
}

func addFinalizer(ctx context.Context, c client.Client, jas *v1alpha1.AutoScaler) error {
	if !utils.ContainsString(jas.ObjectMeta.Finalizers, utils.MeetingFinalizer) {
		jas.ObjectMeta.Finalizers = append(jas.ObjectMeta.Finalizers, utils.MeetingFinalizer)
		return c.Update(ctx, jas)
	}
	return nil
}

func (p *prom) Watch() error {
	ctx, cancel := context.WithTimeout(p.ctx, promRequestTimeout)
	defer cancel()
	p.metricsCounting(ctx)

	return nil
}

func (p *prom) metricsCounting(ctx context.Context) {
	for metric := range p.Spec.Metrics {
		switch p.Spec.Metrics[metric].Resource.Name {
		case v1alpha1.ResourceCPU:
			request := "rate(container_cpu_usage_seconds_total{container=\"jvb\", id=~\"/kubelet.*\"}[5m])" //
			if err := p.updateJVBReplicas(ctx, request, metric); err != nil {
				p.log.Info("can't update jitsi replicas count", "error", err)
			}
			return
		case v1alpha1.ResourceConference:
			request := "jitsi_conferences{job=~\"exporter-jvb-.*\"}" //
			if err := p.updateJVBReplicas(ctx, request, metric); err != nil {
				p.log.Info("can't update jitsi replicas count", "error", err)
			}
			return
		}
	}
}

func (p *prom) updateJVBReplicas(ctx context.Context, request string, m int) error {
	result, _, err := p.apiv1.QueryRange(ctx, request, p.timeRange)
	if err != nil {
		p.log.Info("can't query prometheus", "error", err)
		return err
	}
	var sum model.SampleValue
	for res := range result.(model.Matrix) {
		sum = +result.(model.Matrix)[res].Values[1].Value
	}
	avg := sum / model.SampleValue(len(result.(model.Matrix)))
	if p.Spec.Metrics[m].Resource.TargetAverageUtilization > int32(avg) {
		p.log.Info("scale down")
		jitsi, getErr := p.getJitsiCR()
		if getErr != nil {
			return getErr
		}
		desiredReplicas := jitsi.Spec.JVB.Replicas * (int32(avg) / p.Spec.Metrics[m].Resource.TargetAverageUtilization)
		jitsi.Spec.JVB.Replicas = desiredReplicas
		return p.Client.Update(p.ctx, jitsi)
	}
	p.log.Info("scale up")
	jitsi, getErr := p.getJitsiCR()
	if getErr != nil {
		return getErr
	}
	r := math.RoundToEven(float64(avg) / float64(p.Spec.Metrics[m].Resource.TargetAverageUtilization))
	desiredReplicas := jitsi.Spec.JVB.Replicas * int32(r)
	jitsi.Spec.JVB.Replicas = desiredReplicas
	return p.Client.Update(ctx, jitsi)
}

func (p *prom) getJitsiCR() (*jitsiv1alpha.Jitsi, error) {
	jitsi := &jitsiv1alpha.Jitsi{}
	keyObj := ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: p.Namespace,
		Name:      p.Spec.ScaleTargetRef.Name,
	}}
	if getErr := p.Client.Get(p.ctx, keyObj.NamespacedName, jitsi); getErr != nil {
		return nil, getErr
	}
	return jitsi, nil
}

func (p *prom) Stop() error {
	return nil
}

func (p *influx) Watch() error {
	return nil
}

func (p *influx) Stop() error {
	return nil
}
