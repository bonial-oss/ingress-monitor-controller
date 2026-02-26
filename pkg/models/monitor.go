package models

import (
	"errors"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
)

// ErrMonitorNotFound must be returned by monitor providers if a monitor cannot
// be found.
var ErrMonitorNotFound = errors.New("monitor not found")

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
