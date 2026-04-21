package controller

import (
	"context"
	"testing"
	"time"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func newHTTPRouteSchemeClient(objects ...client.Object) client.Client {
	scheme := fakeclient.NewClientBuilder().Build().Scheme()
	_ = gatewayv1.Install(scheme)
	return fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func TestHTTPRouteReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name        string
		clientFn    func() client.Client
		setup       func(*fake.Service)
		options     config.Options
		req         reconcile.Request
		expected    reconcile.Result
		expectError bool
	}{
		{
			name: "it deletes monitors if httproute was deleted",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "foo",
					Namespace: "default",
				},
			},
			setup: func(s *fake.Service) {
				s.On("DeleteMonitor", matchMonitorSource("foo", "default")).Return(nil)
			},
		},
		{
			name: "it ensures that monitors are present if httproute has annotation",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				},
			},
			clientFn: func() client.Client {
				return newHTTPRouteSchemeClient(&gatewayv1.HTTPRoute{
					TypeMeta: metav1.TypeMeta{
						Kind:       "HTTPRoute",
						APIVersion: "gateway.networking.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "default",
						Annotations: map[string]string{
							config.AnnotationEnabled: "true",
						},
					},
					Spec: gatewayv1.HTTPRouteSpec{
						Hostnames: []gatewayv1.Hostname{"bar.example.com"},
					},
				})
			},
			setup: func(s *fake.Service) {
				s.On("EnsureMonitor", models.MonitorSource{
					Kind:      "HTTPRoute",
					Name:      "bar",
					Namespace: "default",
					Annotations: map[string]string{
						config.AnnotationEnabled: "true",
					},
					URL: "https://bar.example.com",
				}).Return(nil)
			},
		},
		{
			name: "it deletes monitors if httproute does not have annotation",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				},
			},
			clientFn: func() client.Client {
				return newHTTPRouteSchemeClient(&gatewayv1.HTTPRoute{
					TypeMeta: metav1.TypeMeta{
						Kind:       "HTTPRoute",
						APIVersion: "gateway.networking.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "default",
					},
					Spec: gatewayv1.HTTPRouteSpec{
						Hostnames: []gatewayv1.Hostname{"bar.example.com"},
					},
				})
			},
			setup: func(s *fake.Service) {
				s.On("DeleteMonitor", matchMonitorSource("bar", "default")).Return(nil)
			},
		},
		{
			name: "invalid httproute without hostnames is silently ignored",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				},
			},
			clientFn: func() client.Client {
				return newHTTPRouteSchemeClient(&gatewayv1.HTTPRoute{
					TypeMeta: metav1.TypeMeta{
						Kind:       "HTTPRoute",
						APIVersion: "gateway.networking.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "default",
						Annotations: map[string]string{
							config.AnnotationEnabled: "true",
						},
					},
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cl client.Client

			if test.clientFn != nil {
				cl = test.clientFn()
			} else {
				cl = newHTTPRouteSchemeClient()
			}

			svc := &fake.Service{}

			if test.setup != nil {
				test.setup(svc)
			}

			r := NewHTTPRouteReconciler(cl, svc, &test.options)

			result, err := r.Reconcile(context.Background(), test.req)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}

			svc.AssertExpectations(t)
		})
	}
}

func TestHTTPRouteReconciler_Reconcile_DelayCreation(t *testing.T) {
	cl := newHTTPRouteSchemeClient(&gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: "gateway.networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "default",
			Annotations: map[string]string{
				config.AnnotationEnabled: "true",
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{"bar.example.com"},
		},
	})

	r := NewHTTPRouteReconciler(cl, &fake.Service{}, &config.Options{
		CreationDelay: 1 * time.Minute,
	})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "bar",
			Namespace: "default",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	if result.RequeueAfter <= 0 {
		t.Fatalf("expected result.RequeueAfter to be greater than 0, got %s", result.RequeueAfter)
	}
}
