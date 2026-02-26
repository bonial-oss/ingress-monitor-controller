package ingress

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
)

const (
	nginxForceSSLRedirectAnnotation = "nginx.ingress.kubernetes.io/force-ssl-redirect"
)

// Validate checks if ingress fulfills all criteria for an ingress
// monitor and returns an error on any violation. That is, if the ingress
// supports TLS, the TLS hosts must not contain wildcards. The ingress must
// have at least one rule and the rules' host must not contain wildcards (if
// the ingress does not support TLS).
func Validate(ingress *networkingv1.Ingress) error {
	if supportsTLS(ingress) && containsWildcard(ingress.Spec.TLS[0].Hosts[0]) {
		return errors.Errorf("ingress TLS host %q contains wildcards", ingress.Spec.TLS[0].Hosts[0])
	}

	if len(ingress.Spec.Rules) == 0 {
		return errors.New("ingress does not have any rules")
	}

	if containsWildcard(ingress.Spec.Rules[0].Host) {
		return errors.Errorf("ingress host %q contains wildcards", ingress.Spec.Rules[0].Host)
	}

	return nil
}

// BuildMonitorURL builds the url that should be monitored on the ingress.
// Unvalidated ingresses may cause BuildMonitorURL to panic.
func BuildMonitorURL(ingress *networkingv1.Ingress) (string, error) {
	host := buildHostURL(ingress)

	url, err := url.Parse(host)
	if err != nil {
		return "", err
	}

	path, found := ingress.Annotations[config.AnnotationPathOverride]
	if found {
		url.Path = path
	}

	return url.String(), nil
}

func buildHostURL(ingress *networkingv1.Ingress) string {
	if supportsTLS(ingress) {
		return fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	}

	if forceHTTPS(ingress) {
		return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
	}

	return fmt.Sprintf("http://%s", ingress.Spec.Rules[0].Host)
}

func supportsTLS(ingress *networkingv1.Ingress) bool {
	return len(ingress.Spec.TLS) > 0 && len(ingress.Spec.TLS[0].Hosts) > 0 && len(ingress.Spec.TLS[0].Hosts[0]) > 0
}

func forceHTTPS(ingress *networkingv1.Ingress) bool {
	annotations := config.Annotations(ingress.Annotations)

	return annotations.BoolValue(config.AnnotationForceHTTPS) || annotations.BoolValue(nginxForceSSLRedirectAnnotation)
}

func containsWildcard(hostName string) bool {
	return strings.Contains(hostName, "*")
}
