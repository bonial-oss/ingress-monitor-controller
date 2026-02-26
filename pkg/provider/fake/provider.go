package fake

import (
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
	"github.com/stretchr/testify/mock"
)

// Provider is a fake provider that can be used in unit tests.
type Provider struct {
	mock.Mock
}

// Create implements provider.Interface.
func (p *Provider) Create(model *models.Monitor) error {
	args := p.Called(model)

	return args.Error(0)
}

// Create implements provider.Interface.
func (p *Provider) Get(name string) (*models.Monitor, error) {
	args := p.Called(name)
	if obj, ok := args.Get(0).(*models.Monitor); ok {
		return obj, args.Error(1)
	}

	return nil, args.Error(1)
}

// Create implements provider.Interface.
func (p *Provider) Update(model *models.Monitor) error {
	args := p.Called(model)

	return args.Error(0)
}

// Create implements provider.Interface.
func (p *Provider) Delete(name string) error {
	args := p.Called(name)

	return args.Error(0)
}

// GetIPSourceRanges implements provider.Interface.
func (p *Provider) GetIPSourceRanges(model *models.Monitor) ([]string, error) {
	args := p.Called(model)
	if obj, ok := args.Get(0).([]string); ok {
		return obj, args.Error(1)
	}

	return nil, args.Error(1)
}
