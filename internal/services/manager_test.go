package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dev.ict/golang/go-ai/config"
)

// MockService implements Service interface for testing
type MockService struct {
	initCalled bool
	shouldFail bool
}

// Name implements Service.
func (m *MockService) Name() string {
	panic("unimplemented")
}

func (m *MockService) Initialize(ctx context.Context) error {
	m.initCalled = true
	if m.shouldFail {
		return errors.New("mock init failed")
	}
	return nil
}

func TestNewManager(t *testing.T) {
	cfg := &config.Config{}
	mgr := NewManager(cfg)

	assert.NotNil(t, mgr, "Manager should not be nil")
	assert.Equal(t, cfg, mgr.config, "Manager should store config")
	assert.NotNil(t, mgr.services, "Services map should be initialized")
	assert.False(t, mgr.isStarted, "Manager should not be started initially")
}

func TestRegisterService(t *testing.T) {
	mgr := NewManager(&config.Config{})
	svc := &MockService{}

	mgr.RegisterService("test", svc)

	assert.Len(t, mgr.services, 1, "Should have one service registered")
	assert.Equal(t, svc, mgr.services["test"], "Service should be stored with correct name")
}

func TestInitAll(t *testing.T) {
	tests := []struct {
		name      string
		services  map[string]*MockService
		wantError bool
	}{
		{
			name: "successful initialization",
			services: map[string]*MockService{
				"svc1": {shouldFail: false},
				"svc2": {shouldFail: false},
			},
			wantError: false,
		},
		{
			name: "initialization failure",
			services: map[string]*MockService{
				"svc1": {shouldFail: false},
				"svc2": {shouldFail: true},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManager(&config.Config{})

			// Register test services
			for name, svc := range tt.services {
				mgr.RegisterService(name, svc)
			}

			// Test initialization
			err := mgr.InitAll(context.Background())

			if tt.wantError {
				assert.Error(t, err, "InitAll should return error")
				assert.False(t, mgr.isStarted, "Manager should not be marked as started on error")
			} else {
				assert.NoError(t, err, "InitAll should not return error")
				assert.True(t, mgr.isStarted, "Manager should be marked as started")

				// Verify all services were initialized
				for _, svc := range tt.services {
					assert.True(t, svc.initCalled, "Service Initialize should have been called")
				}
			}
		})
	}

	t.Run("double initialization", func(t *testing.T) {
		mgr := NewManager(&config.Config{})
		svc := &MockService{}
		mgr.RegisterService("test", svc)

		// First initialization
		err := mgr.InitAll(context.Background())
		assert.NoError(t, err)

		// Reset mock
		svc.initCalled = false

		// Second initialization
		err = mgr.InitAll(context.Background())
		assert.NoError(t, err)
		assert.False(t, svc.initCalled, "Service should not be initialized twice")
	})
}
