package etherpad

import (
	"context"
	"github.com/onmetal/meeting-operator/apis/etherpad/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/onmetal/meeting-operator/internal/generator/manifests"
)

func (r *Reconciler) make(ctx context.Context, etherpad *v1alpha1.Etherpad) error {
	d := manifests.NewEtherpadTemplate(ctx, etherpad, r.Client, r.Log)
	if err := d.Make(); err != nil {
		r.Log.Info("failed to make deployment", "error", err, "Name", d.Name, "Namespace", d.Namespace)
		return err
	}
	st := manifests.NewEtherpadServiceTemplate(ctx, etherpad, r.Client, r.Log)
	if err := st.Make(); err != nil {
		r.Log.Info("failed to make service", "error", err, "Name", d.Name, "Namespace", d.Namespace)
		return err
	}
	return nil
}

func (r *Reconciler) cleanUpEtherpadObjects(ctx context.Context, e *v1alpha1.Etherpad) error {
	d := manifests.NewEtherpadTemplate(ctx, e, r.Client, r.Log)
	if err := d.Delete(); err != nil && !errors.IsNotFound(err) {
		r.Log.Info("failed to delete deployment", "name", d.Name, "error", err)
		return err
	}
	s := manifests.NewEtherpadServiceTemplate(ctx, e, r.Client, r.Log)
	if err := s.Delete(); err != nil && !errors.IsNotFound(err) {
		r.Log.Info("failed to delete service", "name", s.Name, "error", err)
		return err
	}
	r.Log.Info("web resources were deleted")
	return nil
}
