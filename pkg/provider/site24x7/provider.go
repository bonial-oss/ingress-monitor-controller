package site24x7

import (
	"time"

	site24x7 "github.com/Bonial-International-GmbH/site24x7-go"
	"github.com/Bonial-International-GmbH/site24x7-go/location"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("site24x7-provider")

// Provider manages Site24x7 website monitors.
type Provider struct {
	client           site24x7.Client
	config           config.Site24x7Config
	ipProvider       *location.ProfileIPProvider
	builder          *builder
	sourceRangeCache *cache.Expiring
}

// NewProvider creates a new Site24x7 provider with given Site24x7Config.
func NewProvider(config config.Site24x7Config) *Provider {
	client := site24x7.New(site24x7.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RefreshToken: config.RefreshToken,
	})

	return &Provider{
		client:           client,
		config:           config,
		builder:          newBuilder(client, config.MonitorDefaults),
		sourceRangeCache: cache.NewExpiring(),
	}
}

// Create implements provider.Interface.
func (p *Provider) Create(model *models.Monitor) error {
	monitor, err := p.builder.FromModel(model)
	if err != nil {
		return errors.Wrapf(err, "failed to build site24x7 monitor from model: %#v", model)
	}

	_, err = p.client.Monitors().Create(monitor)
	if err != nil {
		return errors.Wrapf(err, "failed to create site24x7 monitor: %#v", monitor)
	}

	return nil
}

// Create implements provider.Interface.
func (p *Provider) Get(name string) (*models.Monitor, error) {
	monitors, err := p.client.Monitors().List()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list site24x7 monitors")
	}

	for _, monitor := range monitors {
		if monitor.DisplayName != name {
			continue
		}

		m := &models.Monitor{
			ID:   monitor.MonitorID,
			Name: monitor.DisplayName,
			URL:  monitor.Website,
		}

		return m, nil
	}

	return nil, models.ErrMonitorNotFound
}

// Create implements provider.Interface.
func (p *Provider) Update(model *models.Monitor) error {
	monitor, err := p.builder.FromModel(model)
	if err != nil {
		return errors.Wrapf(err, "failed to build site24x7 monitor from model: %#v", model)
	}

	_, err = p.client.Monitors().Update(monitor)
	if err != nil {
		return errors.Wrapf(err, "failed to update site24x7 monitor: %#v", monitor)
	}

	return nil
}

// Create implements provider.Interface.
func (p *Provider) Delete(name string) error {
	monitor, err := p.Get(name)
	if err != nil {
		return err
	}

	err = p.client.Monitors().Delete(monitor.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete site24x7 monitor with ID %s", monitor.ID)
	}

	return nil
}

// getProfileIPProvider lazily creates a ProfileIPProvider. This is an
// optimization to avoid API calls when not needed and also allows us to stub
// out the ProfileIPProvider in tests.
func (p *Provider) getProfileIPProvider() (*location.ProfileIPProvider, error) {
	var err error
	if p.ipProvider == nil {
		p.ipProvider, err = location.NewDefaultProfileIPProvider(p.client)
	}

	return p.ipProvider, err
}

// GetIPSourceRanges implements provider.Interface.
func (p *Provider) GetIPSourceRanges(model *models.Monitor) ([]string, error) {
	monitor, err := p.builder.FromModel(model)
	if err != nil {
		return nil, err
	}

	cachedSourceRanges, ok := p.sourceRangeCache.Get(monitor.LocationProfileID)
	if ok {
		return cachedSourceRanges.([]string), nil
	}

	ipProvider, err := p.getProfileIPProvider()
	if err != nil {
		return nil, err
	}

	locationProfile, err := p.client.LocationProfiles().Get(monitor.LocationProfileID)
	if err != nil {
		return nil, err
	}

	locationIPs, err := ipProvider.GetLocationIPs(locationProfile)
	if err != nil {
		return nil, err
	}

	log.V(1).Info("found ip addresses for location profile", "count", len(locationIPs), "profile-id", locationProfile.ProfileID, "ips", locationIPs)

	sourceRanges := make([]string, len(locationIPs))
	for i, ip := range locationIPs {
		sourceRanges[i] = ip + "/32"
	}

	// Location profiles rarely change so we can just cache them for a day.
	p.sourceRangeCache.Set(monitor.LocationProfileID, sourceRanges, 24*time.Hour)

	return sourceRanges, nil
}
