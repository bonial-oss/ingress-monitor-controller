// Package metrics provides prometheus metric declarations to collect stats
// about monitor creations/updates/deletions.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// MonitorsCreatedTotal is a counter for the total number of successful
	// ingress monitor creation operations.
	MonitorsCreatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ingress_monitor_controller_monitors_created_total",
		Help: "Total number of ingress monitors created by monitor",
	}, []string{"monitor"})

	// MonitorsUpdatedTotal is a counter for the total number of successful
	// ingress monitor update operations.
	MonitorsUpdatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ingress_monitor_controller_monitors_updated_total",
		Help: "Total number of ingress monitors updated by monitor",
	}, []string{"monitor"})

	// MonitorsDeletedTotal is a counter for the total number of successful
	// ingress monitor deletion operations.
	MonitorsDeletedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ingress_monitor_controller_monitors_deleted_total",
		Help: "Total number of ingress monitors deleted by monitor",
	}, []string{"monitor"})

	// IngressValidationErrorsTotal is a counter for the total number of failed
	// ingress validation events. That is: monitor creation was requested for
	// an ingress that was not eligible as a target for an ingress monitor. See
	// the doc of pkg/ingress.Validate for an explanation of the validation
	// rules.
	IngressValidationErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ingress_monitor_controller_ingress_validation_errors_total",
		Help: "Total number of ingress validation errors by namespace and ingress name",
	}, []string{"namespace", "name"})

	// HTTPRouteValidationErrorsTotal is a counter for the total number of
	// failed HTTPRoute validation events.
	HTTPRouteValidationErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ingress_monitor_controller_httproute_validation_errors_total",
		Help: "Total number of HTTPRoute validation errors by namespace and name",
	}, []string{"namespace", "name"})
)

func init() {
	metrics.Registry.MustRegister(
		MonitorsCreatedTotal,
		MonitorsUpdatedTotal,
		MonitorsDeletedTotal,
		IngressValidationErrorsTotal,
		HTTPRouteValidationErrorsTotal,
	)
}
