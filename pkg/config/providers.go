package config

import (
	"io/ioutil"
	"os"

	site24x7api "github.com/Bonial-International-GmbH/site24x7-go/api"
	"sigs.k8s.io/yaml"
)

const (
	// ProviderSite24x7 uses Site24x7 for managing ingress monitors.
	ProviderSite24x7 = "site24x7"

	// ProviderNull does nothing but log create/update/delete monitor events.
	// This is intended for testing purposes only.
	ProviderNull = "null"
)

// ProviderConfig contains the configuration for all supported monitor
// providers.
type ProviderConfig struct {
	Site24x7 Site24x7Config `json:"site24x7"`
}

// Site24x7Config is the configuration for the Site24x7 website monitor
// provider.
type Site24x7Config struct {
	// ClientID is the OAuth2 client ID provided by Site24x7. If not specified,
	// the value will be read from the SITE24X7_CLIENT_ID environment variable.
	ClientID string `json:"clientID"`

	// ClientSecret is the OAuth2 client secret provided by Site24x7. If not
	// specified, the value will be read from the SITE24X7_CLIENT_SECRET
	// environment variable.
	ClientSecret string `json:"clientSecret"`

	// RefreshToken is the OAuth2 refresh token provided by Site24x7. If not
	// specified, the value will be read from the SITE24X7_REFRESH_TOKEN
	// environment variable.
	RefreshToken string `json:"refreshToken"`

	// MonitorDefaults contain defaults that apply to all monitors. The
	// defaults can be overridden explicitly for each monitor via ingress
	// annotations (see annotations.go for all available annotations).
	MonitorDefaults Site24x7MonitorDefaults `json:"monitorDefaults"`
}

// Site24x7MonitorDefaults define the monitor defaults that are used for each
// monitor if not overridden explicitly via ingress annotations.
type Site24x7MonitorDefaults struct {
	// Actions configures default alert actions, where ActionID is the ID of
	// the Site24x7 IT Automation action and AlertType has to be one of the
	// values specified by the Site24x7 action rule constants:
	// https://www.site24x7.com/help/api/#action_constants.
	Actions []site24x7api.ActionRef `json:"actions"`

	// AuthPass sets the default password for endpoints requiring basic auth.
	AuthPass string `json:"authPass"`

	// AuthUser sets the default user for endpoints requiring basic auth.
	AuthUser string `json:"authUser"`

	// AutoLocationProfile configures the behaviour for auto-detecting the
	// location profile to use. If set to true, the first location profile
	// returned by the Site24x7 API will be used. This only applies, if
	// the default LocationProfileID is not set.
	AutoLocationProfile bool `json:"autoLocationProfile"`

	// AutoNotificationProfile configures the behaviour for auto-detecting the
	// notification profile to use. If set to true, the first notification
	// profile returned by the Site24x7 API will be used. This only applies, if
	// the default NotificationProfileID is not set.
	AutoNotificationProfile bool `json:"autoNotificationProfile"`

	// AutoThresholdProfile configures the behaviour for auto-detecting the
	// threshold profile to use. If set to true, the first threshold profile
	// returned by the Site24x7 API will be used. This only applies, if the
	// default ThresholdProfileID is not set.
	AutoThresholdProfile bool `json:"autoThresholdProfile"`

	// AutoMonitorGroup configures the behaviour for auto-detecting the monitor
	// group to use. If set to true, the first monitor group returned by the
	// Site24x7 API will be used. This only applies, if the default
	// MonitorGroupIDs is empty.
	AutoMonitorGroup bool `json:"autoMonitorGroup"`

	// AutoUserGroup configures the behaviour for auto-detecting the user group
	// to use. If set to true, the first user group returned by the Site24x7
	// API will be used. This only applies, if the default UserGroupIDs is
	// empty.
	AutoUserGroup bool `json:"autoUserGroup"`

	// CheckFrequency configures the default check frequency. See
	// https://www.site24x7.com/help/api/#check_interval for a list of valid
	// values.
	CheckFrequency string `json:"checkFrequency"`

	// CustomHeaders configures additional custom HTTP headers to send with
	// each check.
	CustomHeaders []site24x7api.Header `json:"customHeaders"`

	// HTTPMethod sets the default HTTP method to use for all checks. See
	// https://www.site24x7.com/help/api/#http_methods for a list of valid
	// values.
	HTTPMethod string `json:"httpMethod"`

	// LocationProfileID configures the ID of the default location profile used
	// for all checks.
	LocationProfileID string `json:"locationProfileID"`

	// MatchCase configures keyword search. If true, keyword search will be
	// case sensitive.
	MatchCase bool `json:"matchCase"`

	// MonitorGroupIDs configures the default monitor groups. The slice must
	// contain valid monitor group IDs.
	MonitorGroupIDs []string `json:"monitorGroupIDs"`

	// NotificationProfileID configures the ID of the default notification
	// profile used for all checks.
	NotificationProfileID string `json:"notificationProfileID"`

	// ThresholdProfileID configures the ID of the default threshold profile
	// used for all checks.
	ThresholdProfileID string `json:"thresholdProfileID"`

	// Timeout configures the default timeout for connecting to the monitored
	// website. Has to be in range 1-45.
	Timeout int `json:"timeout"`

	// UseNameServer configures whether to resolve DNS or not.
	UseNameServer bool `json:"useNameServer"`

	// UserAgent sets the default user agent string used by all checks.
	UserAgent string `json:"userAgent"`

	// UserGroupIDs configures the default user groups. The slice must contain
	// valid user group IDs.
	UserGroupIDs []string `json:"userGroupIDs"`
}

// NewDefaultProviderConfig creates a new default provider config.
func NewDefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		Site24x7: Site24x7Config{
			ClientID:     os.Getenv("SITE24X7_CLIENT_ID"),
			ClientSecret: os.Getenv("SITE24X7_CLIENT_SECRET"),
			RefreshToken: os.Getenv("SITE24X7_REFRESH_TOKEN"),
			MonitorDefaults: Site24x7MonitorDefaults{
				AutoLocationProfile:     true,
				AutoNotificationProfile: true,
				AutoThresholdProfile:    true,
				AutoMonitorGroup:        true,
				AutoUserGroup:           true,
				CheckFrequency:          "1",
				HTTPMethod:              "G",
				Timeout:                 10,
				UseNameServer:           true,
				CustomHeaders:           []site24x7api.Header{},
				Actions:                 []site24x7api.ActionRef{},
			},
		},
	}
}

// ReadProviderConfig reads the provider configuration from given file.
func ReadProviderConfig(filename string) (*ProviderConfig, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config ProviderConfig

	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
