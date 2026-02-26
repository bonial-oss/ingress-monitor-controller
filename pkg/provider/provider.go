package provider

import (
	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/provider/null"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/provider/site24x7"
	"github.com/pkg/errors"
)

// Interface is the interface for a monitor provider.
type Interface interface {
	// Create creates a monitor based on the given model. Must return an error
	// if the monitor creation fails.
	Create(model *models.Monitor) error

	// Get retrieves a monitor by its name. Must return
	// models.ErrMonitorNotFound if the monitor does not exist.
	Get(name string) (*models.Monitor, error)

	// Update updates a monitor based on the given model. Must return an error
	// if the monitor update fails.
	Update(model *models.Monitor) error

	// Delete delete a monitor by its name. Must return an error if the monitor
	// deletion fails.
	Delete(name string) error

	// GetIPSourceRanges returns a list of CIDR blocks that the provider is
	// performing the monitoring checks from. The source ranges are
	// automatically added to the source range whitelist of the
	// nginx-ingress-controller if an ingress uses whitelisting.
	GetIPSourceRanges(model *models.Monitor) ([]string, error)
}

// New creates a new monitor provider by name. Returns an error if the named
// provider is not supported.
func New(name string, c config.ProviderConfig) (Interface, error) {
	switch name {
	case config.ProviderSite24x7:
		return site24x7.NewProvider(c.Site24x7), nil
	case config.ProviderNull:
		return &null.Provider{}, nil
	default:
		return nil, errors.Errorf("unsupported provider %q", name)
	}
}
