package monitor

import (
	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/ingress"
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
	// EnsureMonitor ensures that a monitor is in sync with the current ingress
	// configuration. If the monitor does not exist, it will be created.
	EnsureMonitor(ingress *networkingv1.Ingress) error

	// DeleteMonitor deletes the monitor for an ingress. It must not be treated
	// as an error if the monitor was already deleted.
	DeleteMonitor(ingress *networkingv1.Ingress) error

	// GetProviderIPSourceRanges retrieves the IP source ranges that the
	// monitor provider is using to perform checks from. It is a list of CIDR
	// blocks. These source ranges can be used to update the IP whitelist (if
	// one is defined) of an ingress to allow checks by the monitor provider.
	GetProviderIPSourceRanges(ingress *networkingv1.Ingress) ([]string, error)

	// AnnotateIngress updates annotations of ingress if needed. If annotations
	// were added, updated or deleted, the return value will be true.
	AnnotateIngress(ingress *networkingv1.Ingress) (updated bool, err error)
}

type service struct {
	provider provider.Interface
	namer    *Namer
	options  *config.Options
}

// NewService creates a new Service with options. Returns an error if service
// initialization fails.
func NewService(options *config.Options) (Service, error) {
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
func (s *service) EnsureMonitor(ing *networkingv1.Ingress) error {
	err := ingress.Validate(ing)
	if err != nil {
		metrics.IngressValidationErrorsTotal.WithLabelValues(ing.Namespace, ing.Name).Inc()
		log.V(1).Info("ignoring unsupported ingress", "namespace", ing.Namespace, "name", ing.Name, "error", err)
		return nil
	}

	newMonitor, err := s.buildMonitorModel(ing)
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
func (s *service) DeleteMonitor(ingress *networkingv1.Ingress) error {
	name, err := s.namer.Name(ingress)
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

func (s *service) buildMonitorModel(ing *networkingv1.Ingress) (*models.Monitor, error) {
	name, err := s.namer.Name(ing)
	if err != nil {
		return nil, err
	}

	url, err := ingress.BuildMonitorURL(ing)
	if err != nil {
		return nil, err
	}

	monitor := &models.Monitor{
		URL:         url,
		Name:        name,
		Annotations: ing.Annotations,
	}

	return monitor, nil
}

// GetProviderIPSourceRanges implements Service.
func (s *service) GetProviderIPSourceRanges(ing *networkingv1.Ingress) ([]string, error) {
	err := ingress.Validate(ing)
	if err != nil {
		metrics.IngressValidationErrorsTotal.WithLabelValues(ing.Namespace, ing.Name).Inc()
		log.V(1).Info("ignoring unsupported ingress", "namespace", ing.Namespace, "name", ing.Name, "error", err)
		return nil, nil
	}

	monitor, err := s.buildMonitorModel(ing)
	if err != nil {
		return nil, err
	}

	return s.provider.GetIPSourceRanges(monitor)
}
