package controller

import (
	"context"
	"testing"
	"time"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// matchIngressWithAnnotations creates a matcher that verifies an ingress has
// the expected name, namespace, and annotations, while ignoring metadata like
// TypeMeta and ResourceVersion that can vary between test runs or dependency versions.
func matchIngressWithAnnotations(name, namespace string, annotations map[string]string) interface{} {
	return mock.MatchedBy(func(ing *networkingv1.Ingress) bool {
		if ing == nil {
			return false
		}
		if ing.Name != name {
			return false
		}
		if ing.Namespace != namespace {
			return false
		}
		// Verify expected annotations are present (allows extra annotations)
		for key, expectedValue := range annotations {
			if ing.Annotations[key] != expectedValue {
				return false
			}
		}
		return true
	})
}

// matchIngress creates a matcher that verifies an ingress has the expected
// name and namespace, without checking annotations.
func matchIngress(name, namespace string) interface{} {
	return mock.MatchedBy(func(ing *networkingv1.Ingress) bool {
		if ing == nil {
			return false
		}
		return ing.Name == name && ing.Namespace == namespace
	})
}

func TestIngressReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name        string
		clientFn    func() client.Client
		setup       func(*fake.Service)
		options     config.Options
		req         reconcile.Request
		expected    reconcile.Result
		expectError bool
		validate    func(*testing.T, client.Client, *fake.Service)
	}{
		{
			name: "it deletes monitors if ingress was deleted",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "foo",
					Namespace: "kube-system",
				},
			},
			setup: func(s *fake.Service) {
				s.On("DeleteMonitor", matchIngress("foo", "kube-system")).Return(nil)
			},
		},
		{
			name: "it ensures that monitors are present if ingress has annotation",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "bar",
					Namespace: "kube-system",
				},
			},
			clientFn: func() client.Client {
				return fakeclient.NewFakeClient(&networkingv1.Ingress{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Ingress",
						APIVersion: "networking.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "kube-system",
						Annotations: map[string]string{
							config.AnnotationEnabled: "true",
						},
					},
				})
			},
			setup: func(s *fake.Service) {
				annotations := map[string]string{
					config.AnnotationEnabled: "true",
				}

				s.On("AnnotateIngress", matchIngressWithAnnotations("bar", "kube-system", annotations)).Return(false, nil)
				s.On("EnsureMonitor", matchIngressWithAnnotations("bar", "kube-system", annotations)).Return(nil)
			},
		},
		{
			name: "it first updates the ingress if it receives annotation update, but does not update the monitor",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "bar",
					Namespace: "kube-system",
				},
			},
			clientFn: func() client.Client {
				return fakeclient.NewFakeClient(&networkingv1.Ingress{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Ingress",
						APIVersion: "networking.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "kube-system",
						Annotations: map[string]string{
							config.AnnotationEnabled: "true",
						},
					},
				})
			},
			setup: func(s *fake.Service) {
				annotations := map[string]string{
					config.AnnotationEnabled: "true",
				}

				s.On("AnnotateIngress", matchIngressWithAnnotations("bar", "kube-system", annotations)).Return(true, nil)
			},
		},
		{
			name: "it deletes monitors if ingress does not have annotation",
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "bar",
					Namespace: "kube-system",
				},
			},
			clientFn: func() client.Client {
				return fakeclient.NewFakeClient(&networkingv1.Ingress{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Ingress",
						APIVersion: "networking.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "kube-system",
					},
				})
			},
			setup: func(s *fake.Service) {
				s.On("DeleteMonitor", matchIngress("bar", "kube-system")).Return(nil)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var client client.Client

			if test.clientFn != nil {
				client = test.clientFn()
			} else {
				client = fakeclient.NewFakeClient()
			}

			svc := &fake.Service{}

			if test.setup != nil {
				test.setup(svc)
			}

			r := NewIngressReconciler(client, svc, &test.options)

			result, err := r.Reconcile(context.Background(), test.req)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}

			if test.validate != nil {
				test.validate(t, client, svc)
			}

			// Verify all expected mock calls were made
			svc.AssertExpectations(t)
		})
	}
}

func TestIngressReconciler_Reconcile_DelayCreation(t *testing.T) {
	client := fakeclient.NewFakeClient(&networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "kube-system",
			Annotations: map[string]string{
				config.AnnotationEnabled: "true",
			},
			CreationTimestamp: metav1.Now(),
		},
	})

	r := NewIngressReconciler(client, &fake.Service{}, &config.Options{
		CreationDelay: 1 * time.Minute,
	})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "bar",
			Namespace: "kube-system",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	if result.RequeueAfter <= 0 {
		t.Fatalf("expected result.RequeueAfter to be greater than 0, got %s", result.RequeueAfter)
	}
}
