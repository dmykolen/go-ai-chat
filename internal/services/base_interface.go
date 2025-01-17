package services

import (
	"context"

	"gitlab.dev.ict/golang/libs/gologgers"
)

type Service interface {
	Name() string
	Initialize(ctx context.Context) error
}

type BaseService struct {
	name   string
	logger *gologgers.Logger
}

func NewBaseService(name string, logger *gologgers.Logger) BaseService {
	return BaseService{name: name, logger: logger}
}

func (s *BaseService) Name() string {
	return s.name
}
