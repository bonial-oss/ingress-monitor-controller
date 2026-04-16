package httproute

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/pkg/errors"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Validate checks if an HTTPRoute fulfills all criteria for monitoring and
// returns an error on any violation. The HTTPRoute must have at least one
// hostname and hostnames must not contain wildcards.
func Validate(route *gatewayv1.HTTPRoute) error {
	if len(route.Spec.Hostnames) == 0 {
		return errors.New("httproute does not have any hostnames")
	}

	hostname := string(route.Spec.Hostnames[0])

	if containsWildcard(hostname) {
		return errors.Errorf("httproute hostname %q contains wildcards", hostname)
	}

	return nil
}

// BuildMonitorURL builds the URL that should be monitored for the HTTPRoute.
// Unvalidated HTTPRoutes may cause BuildMonitorURL to panic.
func BuildMonitorURL(route *gatewayv1.HTTPRoute) (string, error) {
	hostname := string(route.Spec.Hostnames[0])
	host := buildHostURL(hostname, route.Annotations)

	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}

	path, found := route.Annotations[config.AnnotationPathOverride]
	if found {
		u.Path = path
	}

	return u.String(), nil
}

func buildHostURL(hostname string, annotations map[string]string) string {
	a := config.Annotations(annotations)

	if a.BoolValue(config.AnnotationForceHTTP) {
		return fmt.Sprintf("http://%s", hostname)
	}

	return fmt.Sprintf("https://%s", hostname)
}

func containsWildcard(hostname string) bool {
	return strings.Contains(hostname, "*")
}

// NewMonitorSource creates a MonitorSource from an HTTPRoute resource. The
// route must have been validated before calling this function.
func NewMonitorSource(route *gatewayv1.HTTPRoute) (models.MonitorSource, error) {
	monitorURL, err := BuildMonitorURL(route)
	if err != nil {
		return models.MonitorSource{}, err
	}

	return models.MonitorSource{
		Kind:        "HTTPRoute",
		Name:        route.Name,
		Namespace:   route.Namespace,
		Annotations: route.Annotations,
		URL:         monitorURL,
	}, nil
}
