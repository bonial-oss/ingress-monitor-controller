package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Global Annotations.
const (
	// AnnotationEnabled controls whether a monitor is created for an ingress
	// or not.
	AnnotationEnabled = "ingress-monitor.bonial.com/enabled"

	// AnnotationForceHTTPS forces the monitor to use HTTPS if set to "true".
	AnnotationForceHTTPS = "ingress-monitor.bonial.com/force-https"

	// AnnotationForceHTTP forces the monitor to use HTTP instead of HTTPS if
	// set to "true". This is primarily useful for HTTPRoute resources which
	// default to HTTPS.
	AnnotationForceHTTP = "ingress-monitor.bonial.com/force-http"

	// AnnotationPathOverride configures a custom path that should be monitored
	// (e.g. "/health").
	AnnotationPathOverride = "ingress-monitor.bonial.com/path-override"
)

// Site24x7 Provider Annotations.
const (
	// AnnotationSite24x7Actions configures custom alert actions for this
	// monitor. The value has to be a json array. Example:
	//
	//   site24x7.ingress-monitor.bonial.com/actions: |
	//     [{"action_id": "123","alert_type":0},{"action_id": "456","alert_type":1}]
	//
	// where action_id is the ID of the Site24x7 IT Automation action and
	// alert_type has to be one of the values specified by the Site24x7 action
	// rule constants: https://www.site24x7.com/help/api/#action_constants.
	AnnotationSite24x7Actions = "site24x7.ingress-monitor.bonial.com/actions"

	// AnnotationSite24x7AuthPass sets the password if basic auth is required.
	AnnotationSite24x7AuthPass = "site24x7.ingress-monitor.bonial.com/auth-pass"

	// AnnotationSite24x7AuthUser sets the username if basic auth is required.
	AnnotationSite24x7AuthUser = "site24x7.ingress-monitor.bonial.com/auth-user"

	// AnnotationSite24x7CheckFrequency overrides the check frequency. See
	// https://www.site24x7.com/help/api/#check_interval for a list of valid
	// values.
	AnnotationSite24x7CheckFrequency = "site24x7.ingress-monitor.bonial.com/check-frequency"

	// AnnotationSite24x7CustomHeaders configures additional custom HTTP
	// headers to send with each check. The value has to be a json array.
	// Example:
	//
	//   site24x7.ingress-monitor.bonial.com/custom-headers: |
	//     [{"name":"Content-Type","value":"application/json"}]
	//
	AnnotationSite24x7CustomHeaders = "site24x7.ingress-monitor.bonial.com/custom-headers"

	// AnnotationSite24x7HTTPMethod overrides the HTTP method to use for the
	// check. See https://www.site24x7.com/help/api/#http_methods for a list of
	// valid values.
	AnnotationSite24x7HTTPMethod = "site24x7.ingress-monitor.bonial.com/http-method"

	// AnnotationSite24x7LocationProfileID overrides the ID of the location
	// profile used for the check.
	AnnotationSite24x7LocationProfileID = "site24x7.ingress-monitor.bonial.com/location-profile-id"

	// AnnotationSite24x7MatchCase configures keyword search. If "true",
	// keyword search will be case sensitive.
	AnnotationSite24x7MatchCase = "site24x7.ingress-monitor.bonial.com/match-case"

	// AnnotationSite24x7MonitorGroupIDs overrides the monitor groups for this
	// monitor. Expects a comma separated list of monitor group IDs.
	AnnotationSite24x7MonitorGroupIDs = "site24x7.ingress-monitor.bonial.com/monitor-group-ids"

	// AnnotationSite24x7NotificationProfileID overrides the ID of the
	// notification profile used for the check.
	AnnotationSite24x7NotificationProfileID = "site24x7.ingress-monitor.bonial.com/notification-profile-id"

	// AnnotationSite24x7ThresholdProfileID overrides the ID of the threshold
	// profile used for the check.
	AnnotationSite24x7ThresholdProfileID = "site24x7.ingress-monitor.bonial.com/threshold-profile-id"

	// AnnotationSite24x7Timeout overrides the timeout for connecting to the
	// website. Has to be in range 1-45.
	AnnotationSite24x7Timeout = "site24x7.ingress-monitor.bonial.com/timeout"

	// AnnotationSite24x7UseNameServer configures whether to resolve DNS or
	// not. If set to "true", the IP address is resolved using DNS.
	AnnotationSite24x7UseNameServer = "site24x7.ingress-monitor.bonial.com/use-name-server"

	// AnnotationSite24x7UserAgent overrides the default user agent string used
	// by the check.
	AnnotationSite24x7UserAgent = "site24x7.ingress-monitor.bonial.com/user-agent"

	// AnnotationSite24x7UserGroupIDs overrides the user groups for this
	// monitor. Expects a comma separated list of user group IDs.
	AnnotationSite24x7UserGroupIDs = "site24x7.ingress-monitor.bonial.com/user-group-ids"
)

// Annotations is a container for ingress annotations with added functionality
// for parsing and defaulting annotation values.
type Annotations map[string]string

// StringValue returns the string value of an annotation. If the annotations
// does not exist, the optional default value will be returned, empty string
// otherwise.
func (a Annotations) StringValue(name string, defaultValue ...string) string {
	if val, ok := a[name]; ok {
		return val
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return ""
}

// StringSliceValue returns the string slice value of an annotation. The
// annotation value is separated on commas. If the annotations does not exist,
// the optional default value will be returned, nil otherwise.
func (a Annotations) StringSliceValue(name string, defaultValue ...[]string) []string {
	val := a.StringValue(name)
	if len(val) > 0 {
		return strings.Split(val, ",")
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return nil
}

// BoolValue returns the bool value of an annotation. If the annotations does
// not exist, the optional default value will be returned, false otherwise. If
// the annotation's value cannot be parsed as bool, false is returned.
func (a Annotations) BoolValue(name string, defaultValue ...bool) bool {
	if sval, ok := a[name]; ok {
		val, err := strconv.ParseBool(sval)
		if err != nil {
			log.Errorf("invalid bool value in annotation %q: %s", name, sval)
		}

		return val
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return false
}

// IntValue returns the int value of an annotation. If the annotations does not
// exist, the optional default value will be returned, zero otherwise. If the
// annotation's value cannot be parsed as int, zero is returned.
func (a Annotations) IntValue(name string, defaultValue ...int) int {
	if sval, ok := a[name]; ok {
		val, err := strconv.Atoi(sval)
		if err != nil {
			log.Errorf("invalid int value in annotation %q: %s", name, sval)
		}

		return val
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

// ParseJSON parses the value of the annotation into p. P must be a pointer. If
// the annotation does not exist, p is not altered. JSON will return any errors
// occurring during unmarshal operations.
func (a Annotations) ParseJSON(name string, p interface{}) error {
	val, ok := a[name]
	if !ok {
		return nil
	}

	err := json.Unmarshal([]byte(val), p)
	if err != nil {
		return fmt.Errorf("invalid json in annotation %q: %s: %v", name, val, err)
	}

	return nil
}
