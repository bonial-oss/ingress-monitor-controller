package monitor

import (
	"errors"
	"testing"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/provider/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestService_AnnotateIngress(t *testing.T) {
	tests := []struct {
		name        string
		ingress     *networkingv1.Ingress
		expected    bool
		expectedErr error
		setup       func(*fake.Provider)
		validate    func(*testing.T, *networkingv1.Ingress, *fake.Provider)
	}{
		{
			name: "ingress objects without monitor annotation are not updated",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "kube-system",
				},
			},
			expected: false,
			validate: func(t *testing.T, ingress *networkingv1.Ingress, _ *fake.Provider) {
				assert.Nil(t, ingress.Annotations)
			},
		},
		{
			name: `ingress objects with monitor annotation with value "false" are not updated`,
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "kube-system",
					Annotations: map[string]string{
						config.AnnotationEnabled: "false",
					},
				},
			},
			expected: false,
			validate: func(t *testing.T, ingress *networkingv1.Ingress, _ *fake.Provider) {
				assert.Equal(t, ingress.Annotations, map[string]string{
					config.AnnotationEnabled: "false",
				})
			},
		},
		{
			name: `ingress objects without source range whitelist annotation are not updated`,
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "kube-system",
					Annotations: map[string]string{
						config.AnnotationEnabled: "true",
					},
				},
			},
			validate: func(t *testing.T, ingress *networkingv1.Ingress, _ *fake.Provider) {
				assert.Equal(t, ingress.Annotations, map[string]string{
					config.AnnotationEnabled: "true",
				})
			},
		},
		{
			name: `error while retrieving provider source ranges`,
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "kube-system",
					Annotations: map[string]string{
						config.AnnotationEnabled:            "true",
						nginxWhitelistSourceRangeAnnotation: "5.6.7.8/32,1.2.3.4/32,9.10.11.12/32",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "foo.bar.baz"},
					},
				},
			},
			setup: func(p *fake.Provider) {
				p.On("GetIPSourceRanges", mock.Anything).Return(nil, errors.New("whoops"))
			},
			expectedErr: errors.New("whoops"),
		},
		{
			name: `empty provider source ranges do not cause the object to be patched`,
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "kube-system",
					Annotations: map[string]string{
						config.AnnotationEnabled:            "true",
						nginxWhitelistSourceRangeAnnotation: "5.6.7.8/32,1.2.3.4/32,9.10.11.12/32",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "foo.bar.baz"},
					},
				},
			},
			setup: func(p *fake.Provider) {
				p.On("GetIPSourceRanges", mock.Anything).Return(nil, nil)
			},
			validate: func(t *testing.T, ingress *networkingv1.Ingress, _ *fake.Provider) {
				assert.Equal(t, "5.6.7.8/32,1.2.3.4/32,9.10.11.12/32", ingress.Annotations[nginxWhitelistSourceRangeAnnotation])
			},
		},
		{
			name: `provider source ranges are merged with the configured whitelist annotation`,
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "kube-system",
					Annotations: map[string]string{
						config.AnnotationEnabled:            "true",
						nginxWhitelistSourceRangeAnnotation: "1.2.3.4/32",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "foo.bar.baz"},
					},
				},
			},
			setup: func(p *fake.Provider) {
				p.On("GetIPSourceRanges", mock.Anything).Return([]string{"5.6.7.8/32"}, nil)
			},
			expected: true,
			validate: func(t *testing.T, ingress *networkingv1.Ingress, _ *fake.Provider) {
				assert.Equal(t, "1.2.3.4/32,5.6.7.8/32", ingress.Annotations[nginxWhitelistSourceRangeAnnotation])
			},
		},
		{
			name: `already present source ranges are not added again`,
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "kube-system",
					Annotations: map[string]string{
						config.AnnotationEnabled:            "true",
						nginxWhitelistSourceRangeAnnotation: "1.2.3.4/32",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "foo.bar.baz"},
					},
				},
			},
			setup: func(p *fake.Provider) {
				p.On("GetIPSourceRanges", mock.Anything).Return([]string{"5.6.7.8/32", "1.2.3.4/32"}, nil)
			},
			expected: true,
			validate: func(t *testing.T, ingress *networkingv1.Ingress, _ *fake.Provider) {
				assert.Equal(t, "1.2.3.4/32,5.6.7.8/32", ingress.Annotations[nginxWhitelistSourceRangeAnnotation])
			},
		},
		{
			name: `if provider source ranges are already whitelisted, no patch is created`,
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "kube-system",
					Annotations: map[string]string{
						config.AnnotationEnabled:            "true",
						nginxWhitelistSourceRangeAnnotation: "5.6.7.8/32,1.2.3.4/32",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "foo.bar.baz"},
					},
				},
			},
			setup: func(p *fake.Provider) {
				p.On("GetIPSourceRanges", mock.Anything).Return([]string{"5.6.7.8/32"}, nil)
			},
			expected: false,
			validate: func(t *testing.T, ingress *networkingv1.Ingress, _ *fake.Provider) {
				assert.Equal(t, "5.6.7.8/32,1.2.3.4/32", ingress.Annotations[nginxWhitelistSourceRangeAnnotation])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, provider := newTestService(t, &config.Options{})

			if test.setup != nil {
				test.setup(provider)
			}

			ingress := test.ingress

			annotated, err := svc.AnnotateIngress(ingress)
			if test.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, test.expectedErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, annotated)
			}

			if test.validate != nil {
				test.validate(t, ingress, provider)
			}
		})
	}
}
