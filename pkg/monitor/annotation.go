package monitor

import (
	"strings"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/ingress"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor/metrics"
	networkingv1 "k8s.io/api/networking/v1"
)

const nginxWhitelistSourceRangeAnnotation = "nginx.ingress.kubernetes.io/whitelist-source-range"

// AnnotateIngress updates the nginx whitelist source range annotation on the
// ingress with provider IP source ranges if needed. Returns true if the
// ingress annotations were updated.
func (s *service) AnnotateIngress(ing *networkingv1.Ingress) (bool, error) {
	log := log.WithValues("namespace", ing.Namespace, "name", ing.Name)

	if !shouldPatchSourceRangeWhitelist(ing) {
		log.V(1).Info("ingress does not require patching of source range whitelist")
		return false, nil
	}

	err := ingress.Validate(ing)
	if err != nil {
		metrics.IngressValidationErrorsTotal.WithLabelValues(ing.Namespace, ing.Name).Inc()
		log.V(1).Info("ignoring unsupported ingress", "error", err)
		return false, nil
	}

	source, err := ingress.NewMonitorSource(ing)
	if err != nil {
		return false, err
	}

	providerSourceRanges, err := s.GetProviderIPSourceRanges(source)
	if err != nil {
		return false, err
	}

	if len(providerSourceRanges) == 0 {
		log.V(1).Info("no provider source ranges available for ingress")
		return false, nil
	}

	sourceRanges := strings.Split(ing.Annotations[nginxWhitelistSourceRangeAnnotation], ",")

	sourceRanges, updated := mergeProviderSourceRanges(sourceRanges, providerSourceRanges)
	if !updated {
		log.V(1).Info("no source range update needed for ingress")
		return false, nil
	}

	log.Info("patching ingress")

	ing.Annotations[nginxWhitelistSourceRangeAnnotation] = strings.Join(sourceRanges, ",")

	return true, nil
}

// shouldPatchSourceRangeWhitelist returns true if the source range whitelist
// of an ingress should be patched. Patching is necessary if the ingress has a
// monitor enabled and has configured the
// nginx.ingress.kubernetes.io/whitelist-source-range annotation to only allow
// traffic from whitelisted sources.
func shouldPatchSourceRangeWhitelist(ingress *networkingv1.Ingress) bool {
	annotations := config.Annotations(ingress.Annotations)

	if !annotations.BoolValue(config.AnnotationEnabled) {
		return false
	}

	return len(ingress.Annotations[nginxWhitelistSourceRangeAnnotation]) > 0
}

// mergeProviderSourceRanges merges the providerSourceRanges into the source
// ranges that are configured in the ingresses' whitelist and returns the final
// whitelist as slice of strings. It ensures that IP ranges that are already
// present are not added again. The second return value denotes whether the
// source ranges changed (true) or not (false).
func mergeProviderSourceRanges(sourceRanges, providerSourceRanges []string) ([]string, bool) {
	missingSourceRanges := difference(providerSourceRanges, sourceRanges)

	if len(missingSourceRanges) == 0 {
		return sourceRanges, false
	}

	log.Info("missing source ranges", "cidr block", missingSourceRanges)

	sourceRanges = append(sourceRanges, missingSourceRanges...)

	return sourceRanges, true
}

// difference returns elements that are in a but not in b.
func difference(a, b []string) []string {
	seen := make(map[string]struct{}, len(b))

	for _, el := range b {
		seen[el] = struct{}{}
	}

	var diff []string

	for _, el := range a {
		if _, found := seen[el]; !found {
			diff = append(diff, el)
		}
	}

	return diff
}
