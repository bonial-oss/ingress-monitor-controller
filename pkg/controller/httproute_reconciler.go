package controller

import (
	"context"
	"time"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/httproute"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor/metrics"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// HTTPRouteReconciler reconciles HTTPRoute resources to their desired
// monitoring state.
type HTTPRouteReconciler struct {
	client.Client

	monitorService monitor.Service
	creationDelay  time.Duration
}

// NewHTTPRouteReconciler creates a new *HTTPRouteReconciler.
func NewHTTPRouteReconciler(client client.Client, monitorService monitor.Service, options *config.Options) *HTTPRouteReconciler {
	return &HTTPRouteReconciler{
		Client:         client,
		monitorService: monitorService,
		creationDelay:  options.CreationDelay,
	}
}

// Reconcile creates, updates or deletes monitors whenever an HTTPRoute
// changes. It implements reconcile.Reconciler.
func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	route := &gatewayv1.HTTPRoute{}

	err := r.Get(ctx, req.NamespacedName, route)
	if apierrors.IsNotFound(err) {
		source := models.MonitorSource{
			Name:      req.Name,
			Namespace: req.Namespace,
		}

		err = r.monitorService.DeleteMonitor(source)
	} else if err == nil {
		if route.Annotations[config.AnnotationEnabled] == "true" {
			createAfter := time.Until(route.CreationTimestamp.Add(r.creationDelay))

			if createAfter > 0 {
				return reconcile.Result{RequeueAfter: createAfter}, nil
			}

			err = r.handleCreateOrUpdate(route)
		} else {
			source := models.MonitorSource{
				Name:      route.Name,
				Namespace: route.Namespace,
			}

			err = r.monitorService.DeleteMonitor(source)
		}
	}

	return reconcile.Result{}, err
}

func (r *HTTPRouteReconciler) handleCreateOrUpdate(route *gatewayv1.HTTPRoute) error {
	err := httproute.Validate(route)
	if err != nil {
		metrics.HTTPRouteValidationErrorsTotal.WithLabelValues(route.Namespace, route.Name).Inc()
		return nil
	}

	source, err := httproute.NewMonitorSource(route)
	if err != nil {
		return err
	}

	return r.monitorService.EnsureMonitor(source)
}
