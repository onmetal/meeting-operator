package whiteboard

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/onmetal/meeting-operator/apis/whiteboard/v1alpha2"
	meetingerr "github.com/onmetal/meeting-operator/internal/errors"
	"github.com/onmetal/meeting-operator/internal/utils"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type whiteboard struct {
	client.Client
	*v1alpha2.WhiteBoard

	ctx    context.Context
	log    logr.Logger
	labels map[string]string
}

type Service struct {
	client.Client

	ctx             context.Context
	log             logr.Logger
	name, namespace string
	labels          map[string]string
	annotations     map[string]string
	serviceType     v1.ServiceType
	ports           []v1alpha2.Port
}

func newInstance(ctx context.Context, c client.Client,
	l logr.Logger, req ctrl.Request) (WhiteBoard, error) {
	w := &v1alpha2.WhiteBoard{}
	if err := c.Get(ctx, req.NamespacedName, w); err != nil {
		return nil, err
	}
	if !w.DeletionTimestamp.IsZero() {
		return &whiteboard{
			Client:     c,
			WhiteBoard: w,
			ctx:        ctx,
			log:        l,
		}, meetingerr.UnderDeletion()
	}
	if err := utils.AddFinalizer(ctx, c, w); err != nil {
		l.Info("can't add finalizer to etherpad", "error", err)
	}
	labels := utils.GetDefaultLabelsForApp(w.Name)
	return &whiteboard{
		Client:     c,
		WhiteBoard: w,
		ctx:        ctx,
		log:        l,
		labels:     labels,
	}, nil
}

func newService(w *whiteboard) WhiteBoard {
	return &Service{
		Client:      w.Client,
		ports:       w.Spec.Ports,
		serviceType: w.Spec.ServiceType,
		name:        w.Name,
		namespace:   w.Namespace,
		ctx:         w.ctx,
		log:         w.log,
		annotations: w.Spec.ServiceAnnotations,
		labels:      w.labels,
	}
}
