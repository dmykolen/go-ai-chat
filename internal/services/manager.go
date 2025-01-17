package services

import (
	"context"
	"sync"

	"gitlab.dev.ict/golang/go-ai/config"
)

// Manager handles service lifecycle and dependencies
type Manager struct {
	config    *config.Config
	services  map[string]Service
	mu        sync.RWMutex
	isStarted bool
}

// NewManager creates a new service manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:   cfg,
		services: make(map[string]Service),
	}
}

// RegisterService adds a new service to the manager
func (m *Manager) RegisterService(name string, svc Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[name] = svc
}

// StartAll initializes all registered services
func (m *Manager) InitAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isStarted {
		return nil
	}

	for _, svc := range m.services {
		if err := svc.Initialize(ctx); err != nil {
			return err
		}
	}

	m.isStarted = true
	return nil
}
