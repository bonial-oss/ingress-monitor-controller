package controller

import (
	"context"
	"time"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/ingress"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor/metrics"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// IngressService defines the monitor service interface needed by the Ingress
// reconciler.
type IngressService interface {
	monitor.Service

	// AnnotateIngress updates annotations of ingress if needed. If annotations
	// were added, updated or deleted, the return value will be true.
	AnnotateIngress(ingress *networkingv1.Ingress) (updated bool, err error)
}

// IngressReconciler reconciles ingresses to their desired state.
type IngressReconciler struct {
	client.Client

	monitorService IngressService
	creationDelay  time.Duration
}

// NewIngressReconciler creates a new *IngressReconciler.
func NewIngressReconciler(client client.Client, monitorService IngressService, options *config.Options) *IngressReconciler {
	return &IngressReconciler{
		Client:         client,
		monitorService: monitorService,
		creationDelay:  options.CreationDelay,
	}
}

// Reconcile creates, updates or deletes ingress monitors whenever an ingress
// changes. It implements reconcile.Reconciler.
func (r *IngressReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	ing := &networkingv1.Ingress{}

	err := r.Get(ctx, req.NamespacedName, ing)
	if apierrors.IsNotFound(err) {
		// The ingress was deleted. Construct a minimal source for monitor
		// deletion.
		source := models.MonitorSource{
			Name:      req.Name,
			Namespace: req.Namespace,
		}

		err = r.monitorService.DeleteMonitor(source)
	} else if err == nil {
		if ing.Annotations[config.AnnotationEnabled] == "true" {
			createAfter := time.Until(ing.CreationTimestamp.Add(r.creationDelay))

			// If a creation delay was configured, we will requeue the
			// reconciliation until after the creation delay passed.
			if createAfter > 0 {
				return reconcile.Result{RequeueAfter: createAfter}, nil
			}

			err = r.handleCreateOrUpdate(ctx, ing)
		} else {
			source := models.MonitorSource{
				Name:      ing.Name,
				Namespace: ing.Namespace,
			}

			err = r.monitorService.DeleteMonitor(source)
		}
	}

	return reconcile.Result{}, err
}

func (r *IngressReconciler) handleCreateOrUpdate(ctx context.Context, ing *networkingv1.Ingress) error {
	updated, err := r.reconcileAnnotations(ctx, ing)
	if err != nil || updated {
		// In case of an error we return it here to force requeuing of the
		// reconciliation request. If the ingress was updated, we return
		// here as well because the update will cause the creation of a new
		// ingress update event which will be consumed by Reconcile and we
		// want to avoid duplicate execution of the EnsureMonitor logic. This
		// is an optimization to avoid unnecessary API calls to the monitor
		// provider.
		return err
	}

	err = ingress.Validate(ing)
	if err != nil {
		metrics.IngressValidationErrorsTotal.WithLabelValues(ing.Namespace, ing.Name).Inc()
		return nil
	}

	source, err := ingress.NewMonitorSource(ing)
	if err != nil {
		return err
	}

	return r.monitorService.EnsureMonitor(source)
}

// reconcileAnnotations reconciles the ingress annotations, that is, it may
// update the nginx.ingress.kubernetes.io/whitelist-source-range annotation
// with ip source ranges of the monitor provider. If annotations were updated,
// it will update the ingress object on the cluster and return true and the
// first return value. The will effectively cause the creation of a new ingress
// update event which is then picked up by the reconciler.
func (r *IngressReconciler) reconcileAnnotations(ctx context.Context, ingress *networkingv1.Ingress) (updated bool, err error) {
	ingressCopy := ingress.DeepCopy()

	updated, err = r.monitorService.AnnotateIngress(ingressCopy)
	if err != nil || !updated {
		return false, err
	}

	err = r.Update(ctx, ingressCopy)
	if err != nil {
		return false, err
	}

	return true, nil
}
