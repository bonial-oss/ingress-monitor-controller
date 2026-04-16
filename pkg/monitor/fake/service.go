package fake

import (
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/stretchr/testify/mock"
	networkingv1 "k8s.io/api/networking/v1"
)

type Service struct {
	mock.Mock
}

func (s *Service) EnsureMonitor(source models.MonitorSource) error {
	args := s.Called(source)

	return args.Error(0)
}

func (s *Service) DeleteMonitor(source models.MonitorSource) error {
	args := s.Called(source)

	return args.Error(0)
}

func (s *Service) GetProviderIPSourceRanges(source models.MonitorSource) ([]string, error) {
	args := s.Called(source)

	var ips []string
	if arg, ok := args.Get(0).([]string); ok {
		ips = arg
	}

	return ips, args.Error(1)
}

func (s *Service) AnnotateIngress(ingress *networkingv1.Ingress) (updated bool, err error) {
	args := s.Called(ingress)

	return args.Bool(0), args.Error(1)
}
