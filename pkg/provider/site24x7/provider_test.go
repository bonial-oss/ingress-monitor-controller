package site24x7

import (
	"errors"
	"testing"

	site24x7api "github.com/Bonial-International-GmbH/site24x7-go/api"
	"github.com/Bonial-International-GmbH/site24x7-go/fake"
	"github.com/Bonial-International-GmbH/site24x7-go/location"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/cache"
)

func TestProvider_Create(t *testing.T) {
	tests := []struct {
		name     string
		model    *models.Monitor
		config   config.Site24x7Config
		setup    func(*fake.Client)
		validate func(*testing.T, *fake.Client)
		expected error
	}{
		{
			name: "creates monitor",
			model: &models.Monitor{
				Name: "my-monitor",
				URL:  "http://my-monitor",
			},
			setup: func(c *fake.Client) {
				monitor := &site24x7api.Monitor{
					DisplayName: "my-monitor",
					Website:     "http://my-monitor",
					Type:        "URL",
				}
				c.FakeMonitors.On("Create", monitor).Return(monitor, nil)
			},
		},
		{
			name: "do not create monitor if the ingress annotations are invalid",
			model: &models.Monitor{
				Name: "my-monitor",
				URL:  "http://my-monitor",
				Annotations: config.Annotations{
					config.AnnotationSite24x7Actions: "{invalidjson",
				},
			},
			validate: func(t *testing.T, c *fake.Client) {
				assert.Len(t, c.FakeMonitors.Calls, 0)
			},
			expected: errors.New(`failed to build site24x7 monitor from model: &models.Monitor{ID:"", Name:"my-monitor", URL:"http://my-monitor", Annotations:config.Annotations{"site24x7.ingress-monitor.bonial.com/actions":"{invalidjson"}}: invalid json in annotation "site24x7.ingress-monitor.bonial.com/actions": {invalidjson: invalid character 'i' looking for beginning of object key string`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, c := newTestProvider(test.config)

			if test.setup != nil {
				test.setup(c)
			}

			err := p.Create(test.model)
			if test.expected != nil {
				require.Error(t, err)
				assert.Equal(t, test.expected.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			if test.validate != nil {
				test.validate(t, c)
			}
		})
	}
}

func TestProvider_Update(t *testing.T) {
	tests := []struct {
		name     string
		model    *models.Monitor
		config   config.Site24x7Config
		setup    func(*fake.Client)
		validate func(*testing.T, *fake.Client)
		expected error
	}{
		{
			name: "auto discovers profile and group IDs from API if enabled",
			model: &models.Monitor{
				Name: "my-monitor",
				URL:  "http://my-monitor",
			},
			config: config.Site24x7Config{
				MonitorDefaults: config.Site24x7MonitorDefaults{
					AutoLocationProfile:     true,
					AutoNotificationProfile: true,
					AutoThresholdProfile:    true,
					AutoMonitorGroup:        true,
					AutoUserGroup:           true,
				},
			},
			setup: func(c *fake.Client) {
				monitor := &site24x7api.Monitor{
					DisplayName:           "my-monitor",
					Website:               "http://my-monitor",
					Type:                  "URL",
					LocationProfileID:     "123",
					NotificationProfileID: "456",
					ThresholdProfileID:    "789",
					UserGroupIDs:          []string{"012"},
					MonitorGroups:         []string{"345"},
				}
				c.FakeMonitors.On("Update", monitor).Return(monitor, nil)

				c.FakeLocationProfiles.On("List").Return([]*site24x7api.LocationProfile{
					{ProfileID: "123"},
				}, nil)

				c.FakeNotificationProfiles.On("List").Return([]*site24x7api.NotificationProfile{
					{ProfileID: "456"},
				}, nil)

				c.FakeLocationProfiles.On("List").Return([]*site24x7api.LocationProfile{
					{ProfileID: "123"},
				}, nil)

				c.FakeThresholdProfiles.On("List").Return([]*site24x7api.ThresholdProfile{
					{ProfileID: "789"},
				}, nil)

				c.FakeUserGroups.On("List").Return([]*site24x7api.UserGroup{
					{UserGroupID: "012"},
				}, nil)

				c.FakeMonitorGroups.On("List").Return([]*site24x7api.MonitorGroup{
					{GroupID: "345"},
				}, nil)
			},
		},
		{
			name: "it will not override explicitly set profile IDs with auto discovered IDs",
			model: &models.Monitor{
				Name: "my-monitor",
				URL:  "http://my-monitor",
				Annotations: config.Annotations{
					config.AnnotationSite24x7LocationProfileID: "456",
				},
			},
			config: config.Site24x7Config{
				MonitorDefaults: config.Site24x7MonitorDefaults{
					AutoLocationProfile: true,
				},
			},
			setup: func(c *fake.Client) {
				monitor := &site24x7api.Monitor{
					DisplayName:       "my-monitor",
					Website:           "http://my-monitor",
					Type:              "URL",
					LocationProfileID: "456",
				}
				c.FakeMonitors.On("Update", monitor).Return(monitor, nil)
			},
		},
		{
			name: "it will error if auto discovery of profile returns no results",
			model: &models.Monitor{
				Name: "my-monitor",
				URL:  "http://my-monitor",
			},
			config: config.Site24x7Config{
				MonitorDefaults: config.Site24x7MonitorDefaults{
					AutoLocationProfile: true,
				},
			},
			setup: func(c *fake.Client) {
				c.FakeLocationProfiles.On("List").Return(nil, nil)
			},
			expected: errors.New(`failed to build site24x7 monitor from model: &models.Monitor{ID:"", Name:"my-monitor", URL:"http://my-monitor", Annotations:config.Annotations(nil)}: no location profiles configured`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, c := newTestProvider(test.config)

			if test.setup != nil {
				test.setup(c)
			}

			err := p.Update(test.model)
			if test.expected != nil {
				require.Error(t, err)
				assert.Equal(t, test.expected.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			if test.validate != nil {
				test.validate(t, c)
			}
		})
	}
}

func TestProvider_Get(t *testing.T) {
	tests := []struct {
		name        string
		monitorName string
		config      config.Site24x7Config
		setup       func(*fake.Client)
		validate    func(*testing.T, *fake.Client)
		expected    *models.Monitor
		expectedErr error
	}{
		{
			name:        "returns models.ErrMonitorNotFound if monitor is not found",
			monitorName: "my-monitor",
			setup: func(c *fake.Client) {
				monitors := []*site24x7api.Monitor{
					{DisplayName: "some-other-monitor"},
				}
				c.FakeMonitors.On("List").Return(monitors, nil)
			},
			expectedErr: models.ErrMonitorNotFound,
		},
		{
			name:        "returns monitor with name",
			monitorName: "my-monitor",
			setup: func(c *fake.Client) {
				monitors := []*site24x7api.Monitor{
					{
						MonitorID:   "123",
						DisplayName: "my-monitor",
						Website:     "http://my-monitor",
					},
				}
				c.FakeMonitors.On("List").Return(monitors, nil)
			},
			expected: &models.Monitor{
				ID:   "123",
				Name: "my-monitor",
				URL:  "http://my-monitor",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, c := newTestProvider(test.config)

			if test.setup != nil {
				test.setup(c)
			}

			monitor, err := p.Get(test.monitorName)
			if test.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, test.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, monitor)
			}

			if test.validate != nil {
				test.validate(t, c)
			}
		})
	}
}

func TestProvider_Delete(t *testing.T) {
	tests := []struct {
		name        string
		monitorName string
		config      config.Site24x7Config
		setup       func(*fake.Client)
		validate    func(*testing.T, *fake.Client)
		expected    error
	}{
		{
			name:        "returns if monitor is not found",
			monitorName: "my-monitor",
			setup: func(c *fake.Client) {
				c.FakeMonitors.On("List").Return(nil, nil)
			},
			expected: models.ErrMonitorNotFound,
		},
		{
			name:        "deletes monitor",
			monitorName: "my-monitor",
			setup: func(c *fake.Client) {
				c.FakeMonitors.On("List").Return([]*site24x7api.Monitor{
					{MonitorID: "123", DisplayName: "some-other-monitor"},
					{MonitorID: "456", DisplayName: "my-monitor"},
				}, nil)

				c.FakeMonitors.On("Delete", "456").Return(nil)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, c := newTestProvider(test.config)

			if test.setup != nil {
				test.setup(c)
			}

			err := p.Delete(test.monitorName)
			if test.expected != nil {
				require.Error(t, err)
				assert.Equal(t, test.expected.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			if test.validate != nil {
				test.validate(t, c)
			}
		})
	}
}

func TestProvider_GetIPSourceRanges(t *testing.T) {
	tests := []struct {
		name        string
		ipProvider  *location.ProfileIPProvider
		model       *models.Monitor
		setup       func(*fake.Client)
		validate    func(*testing.T, *fake.Client)
		expected    []string
		expectedErr error
	}{
		{
			name: "gets IP source ranges for model",
			ipProvider: &location.ProfileIPProvider{
				IPSource: &location.StaticIPSource{
					LocationIPs: map[string][]string{
						"789": []string{"1.3.3.7", "0.8.1.5"},
						"123": []string{"1.2.3.4", "5.6.7.8"},
						"456": []string{"1.1.1.1", "2.2.2.2"},
					},
				},
				Locations: []*site24x7api.Location{
					{LocationID: "123"},
					{LocationID: "456"},
				},
			},
			model: &models.Monitor{
				ID:   "12345678",
				Name: "foobar",
				URL:  "https://foo.bar.baz",
				Annotations: config.Annotations{
					config.AnnotationSite24x7NotificationProfileID: "1234",
					config.AnnotationSite24x7LocationProfileID:     "456",
					config.AnnotationSite24x7ThresholdProfileID:    "67890",
					config.AnnotationSite24x7MonitorGroupIDs:       "12,34,56,78",
					config.AnnotationSite24x7UserGroupIDs:          "111,222,333",
				},
			},
			setup: func(c *fake.Client) {
				locationProfile := &site24x7api.LocationProfile{
					ProfileID:          "1",
					PrimaryLocation:    "456",
					SecondaryLocations: []string{"123"},
				}

				c.FakeLocationProfiles.On("Get", "456").Return(locationProfile, nil)
			},
			expected: []string{"1.1.1.1/32", "2.2.2.2/32", "1.2.3.4/32", "5.6.7.8/32"},
		},
		{
			name: "does not fail if there are no IP address infos available for a location",
			ipProvider: &location.ProfileIPProvider{
				IPSource: &location.StaticIPSource{
					LocationIPs: map[string][]string{
						"789": []string{"1.3.3.7", "0.8.1.5"},
					},
				},
				Locations: []*site24x7api.Location{
					{LocationID: "123"},
					{LocationID: "456"},
				},
			},
			model: &models.Monitor{
				ID:   "12345678",
				Name: "foobar",
				URL:  "https://foo.bar.baz",
				Annotations: config.Annotations{
					config.AnnotationSite24x7NotificationProfileID: "1234",
					config.AnnotationSite24x7LocationProfileID:     "456",
					config.AnnotationSite24x7ThresholdProfileID:    "67890",
					config.AnnotationSite24x7MonitorGroupIDs:       "12,34,56,78",
					config.AnnotationSite24x7UserGroupIDs:          "111,222,333",
				},
			},
			setup: func(c *fake.Client) {
				locationProfile := &site24x7api.LocationProfile{
					ProfileID:          "1",
					PrimaryLocation:    "456",
					SecondaryLocations: []string{"123"},
				}

				c.FakeLocationProfiles.On("Get", "456").Return(locationProfile, nil)
			},
			expected: []string{},
		},
		{
			name: "returns error if location profile lookup fails",
			ipProvider: &location.ProfileIPProvider{
				IPSource: &location.StaticIPSource{
					LocationIPs: map[string][]string{
						"789": []string{"1.3.3.7", "0.8.1.5"},
					},
				},
				Locations: []*site24x7api.Location{
					{LocationID: "123"},
					{LocationID: "456"},
				},
			},
			model: &models.Monitor{
				ID:   "12345678",
				Name: "foobar",
				URL:  "https://foo.bar.baz",
				Annotations: config.Annotations{
					config.AnnotationSite24x7NotificationProfileID: "1234",
					config.AnnotationSite24x7LocationProfileID:     "456",
					config.AnnotationSite24x7ThresholdProfileID:    "67890",
					config.AnnotationSite24x7MonitorGroupIDs:       "12,34,56,78",
					config.AnnotationSite24x7UserGroupIDs:          "111,222,333",
				},
			},
			setup: func(c *fake.Client) {
				c.FakeLocationProfiles.On("Get", "456").Return(nil, errors.New("whoops"))
			},
			expectedErr: errors.New("whoops"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, c := newTestProvider(config.Site24x7Config{})
			p.ipProvider = test.ipProvider

			if test.setup != nil {
				test.setup(c)
			}

			ips, err := p.GetIPSourceRanges(test.model)
			if test.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, test.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, ips)
			}

			if test.validate != nil {
				test.validate(t, c)
			}
		})
	}
}

func TestProvider_GetIPSourceRanges_Cache(t *testing.T) {
	p, c := newTestProvider(config.Site24x7Config{})
	p.ipProvider = &location.ProfileIPProvider{
		IPSource: &location.StaticIPSource{
			LocationIPs: map[string][]string{
				"789": []string{"1.3.3.7", "0.8.1.5"},
				"123": []string{"1.2.3.4", "5.6.7.8"},
				"456": []string{"1.1.1.1", "2.2.2.2"},
			},
		},
		Locations: []*site24x7api.Location{
			{LocationID: "123"},
			{LocationID: "456"},
		},
	}

	locationProfile := &site24x7api.LocationProfile{
		ProfileID:          "1",
		PrimaryLocation:    "456",
		SecondaryLocations: []string{"123"},
	}

	// Only expect one API call to fetch the location profile
	c.FakeLocationProfiles.On("Get", "456").Return(locationProfile, nil).Once()

	model := &models.Monitor{
		ID:   "12345678",
		Name: "foobar",
		URL:  "https://foo.bar.baz",
		Annotations: config.Annotations{
			config.AnnotationSite24x7NotificationProfileID: "1234",
			config.AnnotationSite24x7LocationProfileID:     "456",
			config.AnnotationSite24x7ThresholdProfileID:    "67890",
			config.AnnotationSite24x7MonitorGroupIDs:       "12,34,56,78",
			config.AnnotationSite24x7UserGroupIDs:          "111,222,333",
		},
	}

	expected := []string{"1.1.1.1/32", "2.2.2.2/32", "1.2.3.4/32", "5.6.7.8/32"}

	ips, err := p.GetIPSourceRanges(model)
	require.NoError(t, err)
	require.Equal(t, expected, ips)

	ips2, err := p.GetIPSourceRanges(model)
	require.NoError(t, err)
	require.Equal(t, ips, ips2)
}

func newTestProvider(config config.Site24x7Config) (*Provider, *fake.Client) {
	client := fake.NewClient()

	provider := &Provider{
		client:           client,
		config:           config,
		builder:          newBuilder(client, config.MonitorDefaults),
		sourceRangeCache: cache.NewExpiring(),
	}

	return provider, client
}
