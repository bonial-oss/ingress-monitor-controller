package main

import (
	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/controller"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func setupHTTPRouteController(mgr manager.Manager, svc monitor.Service, options *config.Options) error {
	err := gatewayv1.Install(mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to register gateway API scheme")
	}

	reconciler := controller.NewHTTPRouteReconciler(mgr.GetClient(), svc, options)

	err = builder.
		ControllerManagedBy(mgr).
		Named("httproute-monitor-controller").
		For(&gatewayv1.HTTPRoute{}).
		Complete(reconciler)
	if err != nil {
		return err
	}

	return nil
}
