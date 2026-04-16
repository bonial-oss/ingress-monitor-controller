package httproute

import (
	"testing"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		route    *gatewayv1.HTTPRoute
		expected error
	}{
		{
			name: "valid httproute with hostname",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{"foo.bar.baz"},
				},
			},
		},
		{
			name: "wildcard hostnames are not supported",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{"*.bar.baz"},
				},
			},
			expected: errors.New(`httproute hostname "*.bar.baz" contains wildcards`),
		},
		{
			name:     "httproute needs to have at least one hostname",
			route:    &gatewayv1.HTTPRoute{},
			expected: errors.New(`httproute does not have any hostnames`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.route)
			if test.expected != nil {
				require.Error(t, err)
				assert.Equal(t, test.expected.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildMonitorURL(t *testing.T) {
	tests := []struct {
		name     string
		route    *gatewayv1.HTTPRoute
		expected string
	}{
		{
			name: "defaults to https",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{"foo.bar.baz"},
				},
			},
			expected: "https://foo.bar.baz",
		},
		{
			name: "force http via annotation",
			route: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						config.AnnotationForceHTTP: "true",
					},
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{"foo.bar.baz"},
				},
			},
			expected: "http://foo.bar.baz",
		},
		{
			name: "respect path override annotation",
			route: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						config.AnnotationPathOverride: "/health",
					},
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{"foo.bar.baz"},
				},
			},
			expected: "https://foo.bar.baz/health",
		},
		{
			name: "force http with path override",
			route: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						config.AnnotationForceHTTP:    "true",
						config.AnnotationPathOverride: "health",
					},
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{"foo.bar.baz"},
				},
			},
			expected: "http://foo.bar.baz/health",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, err := BuildMonitorURL(test.route)
			require.NoError(t, err)
			assert.Equal(t, test.expected, url)
		})
	}
}

func TestNewMonitorSource(t *testing.T) {
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-route",
			Namespace: "default",
			Annotations: map[string]string{
				config.AnnotationEnabled: "true",
			},
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{"app.example.com"},
		},
	}

	source, err := NewMonitorSource(route)
	require.NoError(t, err)
	assert.Equal(t, "my-route", source.Name)
	assert.Equal(t, "default", source.Namespace)
	assert.Equal(t, "https://app.example.com", source.URL)
	assert.Equal(t, "true", source.Annotations[config.AnnotationEnabled])
}
