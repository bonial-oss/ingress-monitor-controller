package models

import (
	"errors"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
)

// ErrMonitorNotFound must be returned by monitor providers if a monitor cannot
// be found.
var ErrMonitorNotFound = errors.New("monitor not found")

// MonitorSource is a resource-agnostic representation of a Kubernetes resource
// (e.g. Ingress or HTTPRoute) that serves as input for creating monitors.
type MonitorSource struct {
	// Kind is the kind of the Kubernetes resource (e.g. "Ingress" or
	// "HTTPRoute"). This can be used in name templates to disambiguate
	// resources.
	Kind string

	// Name is the name of the Kubernetes resource.
	Name string

	// Namespace is the namespace of the Kubernetes resource.
	Namespace string

	// Annotations are the annotations on the Kubernetes resource.
	Annotations map[string]string

	// URL is the pre-built monitor URL derived from the resource spec.
	URL string
}

// Monitor is a container for a website monitor.
type Monitor struct {
	// ID is the provider specific ID of a monitor.
	ID string

	// Name is the display name of the monitor.
	Name string

	// URL is the url that the monitor supervises.
	URL string

	// Annotations are the annotations that are attached to the ingress object.
	// These can be used by providers to set custom provider specific
	// configuration.
	Annotations config.Annotations
}
