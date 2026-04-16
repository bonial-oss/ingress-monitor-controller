package monitor

import (
	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor/metrics"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/provider"
	networkingv1 "k8s.io/api/networking/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("monitor-service")

// Service defines the interface for a service that takes care of creating,
// updating or deleting monitors.
type Service interface {
	// EnsureMonitor ensures that a monitor is in sync with the given source.
	// If the monitor does not exist, it will be created.
	EnsureMonitor(source models.MonitorSource) error

	// DeleteMonitor deletes the monitor for the given source. It must not be
	// treated as an error if the monitor was already deleted.
	DeleteMonitor(source models.MonitorSource) error
}

// IngressService extends Service with Ingress-specific functionality for
// managing provider IP source range whitelisting on Ingress resources.
type IngressService interface {
	Service

	// GetProviderIPSourceRanges retrieves the IP source ranges that the
	// monitor provider is using to perform checks from. It is a list of CIDR
	// blocks.
	GetProviderIPSourceRanges(source models.MonitorSource) ([]string, error)

	// AnnotateIngress updates annotations of ingress if needed. If
	// annotations were added, updated or deleted, the return value will be
	// true.
	AnnotateIngress(ingress *networkingv1.Ingress) (updated bool, err error)
}

type service struct {
	provider provider.Interface
	namer    *Namer
	options  *config.Options
}

// NewService creates a new Service with options. Returns an error if service
// initialization fails.
func NewService(options *config.Options) (IngressService, error) {
	provider, err := provider.New(options.ProviderName, options.ProviderConfig)
	if err != nil {
		return nil, err
	}

	namer, err := NewNamer(options.NameTemplate)
	if err != nil {
		return nil, err
	}

	s := &service{
		provider: provider,
		namer:    namer,
		options:  options,
	}

	return s, nil
}

// EnsureMonitor implements Service.
func (s *service) EnsureMonitor(source models.MonitorSource) error {
	newMonitor, err := s.buildMonitorModel(source)
	if err != nil {
		return err
	}

	oldMonitor, err := s.provider.Get(newMonitor.Name)
	if err == models.ErrMonitorNotFound {
		return s.createMonitor(newMonitor)
	} else if err != nil {
		return err
	}

	return s.updateMonitor(oldMonitor, newMonitor)
}

// DeleteMonitor implements Service.
func (s *service) DeleteMonitor(source models.MonitorSource) error {
	name, err := s.namer.Name(source)
	if err != nil {
		return err
	}

	if s.options.NoDelete {
		log.V(1).Info("monitor deletion is disabled, not deleting", "monitor", name)
		return nil
	}

	return s.deleteMonitor(name)
}

func (s *service) createMonitor(monitor *models.Monitor) error {
	err := s.provider.Create(monitor)
	if err != nil {
		return err
	}

	metrics.MonitorsCreatedTotal.WithLabelValues(monitor.Name).Inc()
	log.Info("monitor created", "monitor", monitor.Name)

	return nil
}

func (s *service) updateMonitor(oldMonitor, newMonitor *models.Monitor) error {
	newMonitor.ID = oldMonitor.ID

	err := s.provider.Update(newMonitor)
	if err != nil {
		return err
	}

	metrics.MonitorsUpdatedTotal.WithLabelValues(newMonitor.Name).Inc()
	log.Info("monitor updated", "monitor", newMonitor.Name)

	return nil
}

func (s *service) deleteMonitor(name string) error {
	err := s.provider.Delete(name)
	if err == models.ErrMonitorNotFound {
		log.V(1).Info("monitor is not present", "monitor", name)
		return nil
	} else if err != nil {
		return err
	}

	metrics.MonitorsDeletedTotal.WithLabelValues(name).Inc()
	log.Info("monitor deleted", "monitor", name)

	return nil
}

func (s *service) buildMonitorModel(source models.MonitorSource) (*models.Monitor, error) {
	name, err := s.namer.Name(source)
	if err != nil {
		return nil, err
	}

	monitor := &models.Monitor{
		URL:         source.URL,
		Name:        name,
		Annotations: source.Annotations,
	}

	return monitor, nil
}

// GetProviderIPSourceRanges implements IngressService.
func (s *service) GetProviderIPSourceRanges(source models.MonitorSource) ([]string, error) {
	monitor, err := s.buildMonitorModel(source)
	if err != nil {
		return nil, err
	}

	return s.provider.GetIPSourceRanges(monitor)
}
