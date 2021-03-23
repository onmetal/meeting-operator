package jitsi

import (
	"context"

	jitsiv1alpha1 "github.com/onmetal/meeting-operator/api/v1alpha1"
	"github.com/onmetal/meeting-operator/internal/generator/manifests"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (r *Reconciler) makeJicofo(ctx context.Context, jitsi *jitsiv1alpha1.Jitsi) error {
	d := manifests.NewJitsiTemplate(ctx, "jicofo", jitsi, r.Client, r.Log)
	if err := d.Make(); err != nil {
		r.Log.Info("failed to make deployment", "error", err, "Name", d.Name, "Namespace", d.Namespace)
		return err
	}

	svc := manifests.NewJitsiServiceTemplate(ctx, "jicofo", jitsi, r.Client, r.Log)
	if err := svc.Make(); err != nil {
		r.Log.Info("failed to make service", "error", err, "Name", d.Name, "Namespace", d.Namespace)
		return err
	}
	return nil
}

func (r *Reconciler) cleanupJicofoObjects(ctx context.Context, jitsi *jitsiv1alpha1.Jitsi) error {
	d := manifests.NewJitsiTemplate(ctx, "jicofo", jitsi, r.Client, r.Log)
	if err := d.Delete(); err != nil && !errors.IsNotFound(err) {
		r.Log.Info("failed to delete deployment", "name", d.Name, "error", err)
		return err
	}
	s := manifests.NewJitsiServiceTemplate(ctx, "jicofo", jitsi, r.Client, r.Log)
	if err := s.Delete(); err != nil && !errors.IsNotFound(err) {
		r.Log.Info("failed to delete service", "name", s.Name, "error", err)
		return err
	}
	r.Log.Info("jicofo resources were deleted")
	return nil
}
